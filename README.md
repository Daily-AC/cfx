# cfx

`cfx` is an unofficial Codeforces training CLI for humans and coding agents.
It turns a Codeforces problem into a reproducible local workspace containing
the statement, samples, source template, agent coaching policy, and attempt
history.

`cfx` has no OpenCLI dependency, hosted relay, or telemetry. It talks directly
to the Codeforces public API. Browser-only operations use either an isolated
Chrome profile or the optional extension embedded in the `cfx` binary.

## Features

- Sync and cache the public Codeforces problem and contest catalogs.
- Pick problems by rating, tag, and accepted-history exclusion.
- Search problems by name, ID, tags, rating range, contest, and solved state.
- Search contests by name or sponsor plus year, division, phase, and type.
- Inspect a contest or create a complete multi-problem training workspace.
- Fetch statements and samples despite Codeforces browser challenges by using
  a user-owned persistent Chrome profile.
- Generate `AGENTS.md` so a coding agent coaches progressively instead of
  immediately revealing the solution.
- Compile and run C++23 or Python solutions against samples with timeout and
  whitespace-aware comparison.
- Record tests, submissions, and verdicts in `session.json`.
- Submit only after local samples pass and the user provides `--yes`.
- Emit structured JSON for agent integration.
- Reuse an existing signed-in Chrome profile without copying cookies.

## Requirements

- Chrome or Chromium
- A C++23 compiler for C++ workspaces
- Python 3 for Python workspaces

Prebuilt binaries are attached to tagged GitHub releases for macOS, Linux, and
Windows on amd64 and arm64.

Install from source with Go 1.25 or newer:

```bash
go install github.com/e0-7/cfx/cmd/cfx@latest
```

Or build this checkout:

```bash
go build -o cfx ./cmd/cfx
```

## Quick start

```bash
cfx doctor
cfx extension install
cfx auth
cfx sync

# Search by name and multiple conditions. Flags may appear before or after the query.
cfx problem search "shortest path" --min-rating 1400 --max-rating 1800 \
  --tag graphs --tag "shortest paths" --unsolved

# Find sponsored contests and inspect one.
cfx contest search Huawei --year 2024 --phase FINISHED
cfx contest show 2015

# Fetch every problem into ./cf-contest-4/{A,B,C,D}.
cfx contest start 4

# Pick an unsolved implementation problem around 1200.
cfx pick --rating 1200 --tag implementation --unsolved

# Create ./cf-1795b with statement, samples, main.cpp and AGENTS.md.
cfx start 1795B
cd cf-1795b

cfx show
cfx test
cfx submit --yes
cfx status --wait
```

`cfx extension install` extracts the bundled extension into the user config
directory and opens `chrome://extensions`. Enable Developer mode, choose
**Load unpacked**, and select the path printed by the command. This switches
`cfx` to extension mode and reuses the signed-in Chrome profile where you load
it. The CLI never copies or prints cookies.

Run `cfx extension install` again after upgrading `cfx`. If the extension is
already present, click **Reload** on its card so Chrome picks up the new bundled
files. Browser-backed commands open a short-lived localhost page to wake the
Manifest V3 worker automatically; the page closes itself after connecting.

To use an isolated profile instead:

```bash
cfx config set browserMode dedicated
cfx auth
```

Dedicated mode opens a visible Chrome profile stored under `CFX_HOME`. Close
that Chrome window before launching another browser-backed `cfx` command. If a
profile lock is already held, `cfx` reports this directly and suggests the
extension mode.

## Agent workflow

Run your coding agent from the generated problem directory. The generated
`AGENTS.md` establishes these defaults:

- ask for and review the user's current idea first;
- provide the smallest useful hint before a full solution;
- challenge complexity and edge cases with concrete examples;
- edit the source only when implementation is explicitly requested;
- require real `cfx test --json` evidence before claiming samples pass;
- submit only after the user explicitly requests the external action.

All machine-facing commands support `--json` where structured output is useful:

```text
cfx auth
cfx extension install|path
cfx config get|set <key> [value]
cfx doctor [--json]
cfx sync [--json]
cfx pick [--rating N] [--min-rating N] [--max-rating N] [--tag TAG] [--unsolved] [--json]
cfx problem search [QUERY] [--min-rating N] [--max-rating N] [--contest ID] [--rated] [--tag TAG]... [--unsolved] [--sort solved|rating|name] [--limit N] [--json]
cfx contest search [QUERY] [--year N] [--div N] [--phase PHASE] [--type TYPE] [--limit N] [--json]
cfx contest show <contest-id> [--json]
cfx contest start <contest-id> [--dir PATH] [--lang cpp|python] [--json]
cfx start <contest><index> [--dir PATH] [--lang cpp|python] [--json]
cfx show [--json]
cfx test [--json]
cfx status [--wait] [--json]
cfx submit --yes [--json]
```

Search terms are case-insensitive and all terms must match. Repeated `--tag`
flags are also ANDed. Contest sponsor search uses the public contest name;
`--year` is derived from the UTC start time, and `--div` matches divisions in
that name. `--limit 0` returns all matches.

`cfx contest start` fetches every public programming problem before creating
the destination, then writes this layout atomically:

```text
cf-contest-4/
  contest.json
  README.md
  A/              # normal cfx problem workspace
  B/
  C/
  D/
```

Run `cfx show`, `cfx test`, `cfx submit`, and `cfx status` from an individual
problem directory.

## Architecture

Codeforces exposes catalog and verdict data through its public API but does not
provide problem statements or submission through that API. `cfx` therefore
uses two deliberately separate transports:

1. Public JSON API for problem and contest catalogs, accepted history, and verdicts. Requests are
   rate-limited to one every two seconds and retried only for transport or 5xx
   failures.
2. A visible local Chrome/Chromium session for authentication, statements, and
   the submission form. Dedicated mode launches a persistent profile through
   the Chrome DevTools Protocol. Extension mode uses an unpacked Manifest V3
   extension embedded in the binary and a short-lived bridge bound only to
   `127.0.0.1:27121`.

The bridge exists only while a browser-backed command is waiting. Requests must
include a random installation token. Chrome host-permission requests may omit
the `Origin` header; when present, it must use the `chrome-extension://` scheme.
There is no remote service.

The default Codeforces runtimes are GNU G++23 (`programTypeId=91`) for C++ and
PyPy 3-64 (`programTypeId=70`) for Python. Override the runtime if Codeforces
changes its list:

```bash
cfx config set programTypeId ID
```

Set `CFX_HOME` to isolate config, cache, and the Chrome profile during testing.

## Safety and project status

This project is unofficial and is not endorsed by Codeforces. Use it for your
own training and respect Codeforces rules, especially during live or virtual
contests. `cfx` does not bypass login challenges: authentication is always
completed by the user in a visible browser.

Training workspaces execute local source code. Do not run untrusted workspaces.
See [SECURITY.md](SECURITY.md) for the local trust model and
[CONTRIBUTING.md](CONTRIBUTING.md) for development requirements.

## License

MIT
