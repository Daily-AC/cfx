const BRIDGE_URL = "http://127.0.0.1:27121";
const BRIDGE_TOKEN = "CFX_EXTENSION_TOKEN";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForTab(tabId, timeoutMs = 60000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const tab = await chrome.tabs.get(tabId);
    if (tab.status === "complete") return tab;
    await delay(250);
  }
  throw new Error("Timed out waiting for Codeforces tab to load");
}

async function execute(tabId, func, args = []) {
  const results = await chrome.scripting.executeScript({
    target: { tabId },
    func,
    args,
  });
  return results[0] && results[0].result;
}

async function openCodeforces(url, active) {
  const tab = await chrome.tabs.create({ url, active });
  await waitForTab(tab.id);
  return tab;
}

async function whoami() {
  const tab = await openCodeforces("https://codeforces.com/", true);
  const result = await execute(tab.id, () => {
    const link = document.querySelector('.lang-chooser a[href^="/profile/"]');
    if (!link) {
      throw new Error("Codeforces is not logged in in this Chrome profile; sign in on the tab that cfx opened and retry");
    }
    return link.getAttribute("href").slice("/profile/".length);
  });
  await chrome.tabs.remove(tab.id);
  return { handle: result };
}

async function statement(payload) {
  const url = `https://codeforces.com/problemset/problem/${payload.contestId}/${payload.index}?locale=en`;
  const tab = await openCodeforces(url, false);
  const html = await execute(tab.id, () => {
    const root = document.querySelector(".problem-statement");
    if (!root) {
      throw new Error("Problem statement did not load; complete any visible browser challenge and retry");
    }
    return root.outerHTML;
  });
  await chrome.tabs.remove(tab.id);
  return { html, url };
}

async function submit(payload) {
  const url = `https://codeforces.com/contest/${payload.contestId}/submit`;
  const tab = await openCodeforces(url, true);
  const prepared = await execute(tab.id, (job) => {
    const form = document.querySelector("form.submitForm, form.submit-form");
    if (!form) {
      if (location.pathname.startsWith("/enter")) {
        throw new Error("Codeforces login required in this Chrome profile");
      }
      throw new Error("Codeforces submission form was not found");
    }

    const problem = form.querySelector('[name="submittedProblemIndex"], [name="problemIndex"]');
    const language = form.querySelector('[name="programTypeId"]');
    if (!problem || !language) throw new Error("Codeforces submission fields changed");
    problem.value = job.index;
    language.value = job.programTypeId;
    problem.dispatchEvent(new Event("change", { bubbles: true }));
    language.dispatchEvent(new Event("change", { bubbles: true }));

    const textarea = form.querySelector('textarea[name="source"]');
    const fileInput = form.querySelector('input[type="file"][name="sourceFile"]');
    if (textarea) {
      textarea.value = job.source;
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    } else if (fileInput) {
      const transfer = new DataTransfer();
      transfer.items.add(new File([job.source], "main.cpp", { type: "text/plain" }));
      fileInput.files = transfer.files;
      fileInput.dispatchEvent(new Event("change", { bubbles: true }));
    } else {
      throw new Error("Codeforces source field changed");
    }

    if (typeof form.requestSubmit === "function") form.requestSubmit();
    else form.submit();
    return true;
  }, [payload]);
  if (!prepared) throw new Error("Failed to prepare Codeforces submission");

  const deadline = Date.now() + 30000;
  while (Date.now() < deadline) {
    const current = await chrome.tabs.get(tab.id);
    if (current.url && !current.url.includes("/submit")) {
      return { url: current.url };
    }
    await delay(500);
  }
  throw new Error("Codeforces stayed on the submission page; inspect the visible tab for an error");
}

async function runJob(job) {
  if (job.type === "whoami") return whoami();
  if (job.type === "statement") return statement(job.payload);
  if (job.type === "submit") return submit(job.payload);
  throw new Error(`Unknown cfx job type: ${job.type}`);
}

async function postResult(id, ok, data, error) {
  await fetch(`${BRIDGE_URL}/v1/result`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Cfx-Token": BRIDGE_TOKEN },
    body: JSON.stringify({ id, ok, data, error }),
  });
}

async function poll() {
	if (poll.running) return;
	poll.running = true;
  for (;;) {
    try {
      const response = await fetch(`${BRIDGE_URL}/v1/job`, {
        headers: { "X-Cfx-Token": BRIDGE_TOKEN },
      });
      if (response.status === 204) {
        await delay(1000);
        continue;
      }
      if (!response.ok) throw new Error(`Bridge returned HTTP ${response.status}`);
      const job = await response.json();
      try {
        const data = await runJob(job);
        await postResult(job.id, true, data, "");
      } catch (error) {
        await postResult(job.id, false, null, String(error && error.message || error));
      }
    } catch (_) {
      await delay(1000);
    }
  }
}

chrome.runtime.onMessage.addListener((message, sender) => {
  if (message && message.type === "wake") {
    poll();
    if (sender.tab && sender.tab.id) chrome.tabs.remove(sender.tab.id);
  }
});

chrome.action.onClicked.addListener(() => poll());

poll();
