# cfx

`cfx` is a Codeforces training CLI designed to be driven by both a human and a
coding agent. It keeps problem statements, samples, attempts, and session state
in a local workspace while using Codeforces' public API for catalog and verdict
data.

## Install

From this checkout:

```bash
go build -o cfx ./cmd/cfx
```

The default Codeforces runtimes are GNU G++23 (`programTypeId=91`) for C++ and
PyPy 3-64 (`programTypeId=70`) for Python. Override either with
`cfx config set programTypeId ID` if Codeforces changes its runtime list.

## First run

```bash
./cfx config set handle YOUR_HANDLE
./cfx doctor
./cfx auth
./cfx sync
```

`auth` opens a dedicated Chrome profile under the user config directory. Log in
to Codeforces and return to the terminal. Credentials are never handled or
stored by `cfx`; Chrome owns the session cookies.

## Training loop

```bash
# Pick an unsolved implementation problem around 1200.
./cfx pick --rating 1200 --tag implementation --unsolved

# Fetch the statement and create ./cf-4A with main.cpp, samples and AGENTS.md.
./cfx start 4A
cd cf-4A

# Commands are stable for agent use and accept --json.
cfx show
cfx test
cfx status

# External side effect: --yes is deliberately required.
cfx submit --yes
```

The generated `AGENTS.md` asks the coding agent to coach in stages: clarify the
idea first, provide a small hint on request, review pseudocode, and only write a
complete solution when explicitly asked. `session.json` records local test and
submission attempts so a later agent can resume with evidence instead of chat
memory.

## Commands

```text
cfx auth
cfx config get|set <key> [value]
cfx doctor [--json]
cfx sync [--json]
cfx pick [--rating N] [--min-rating N] [--max-rating N] [--tag TAG] [--unsolved] [--json]
cfx start <contest><index> [--dir PATH] [--lang cpp|python] [--json]
cfx show [--json]
cfx test [--json]
cfx status [--wait] [--json]
cfx submit --yes [--json]
```

Codeforces documents a limit of one API call per two seconds. `cfx` caches the
catalog and avoids polling faster than that limit.
