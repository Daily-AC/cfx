# Contributing

Contributions are welcome. Keep changes focused and include tests for behavior
that can regress.

## Development

Requirements:

- Go 1.25 or newer
- Chrome or Chromium for browser-backed integration tests
- A C++23 compiler and Python 3 for runner tests

Run the local checks before opening a pull request:

```bash
go test -race ./...
go vet ./...
go build ./cmd/cfx
```

Tests must not submit solutions, require a maintainer's Codeforces account, or
store session cookies in the repository. API fixtures should be reduced copies
of real Codeforces responses with personal data removed.

## Integration changes

Codeforces' public API and browser forms are separate integration surfaces.
When changing either one:

1. Add or update a fixture-based regression test.
2. Run a real read-only request against Codeforces.
3. For submission changes, test with an intentionally accepted practice
   solution and report the resulting submission ID in the pull request.

Search and contest changes should also be checked against live
`problemset.problems` and `contest.list?gym=false` responses. A complete
`contest start` integration check must verify the root `contest.json` and at
least one generated problem's statement and sample files.

Never automate login challenges or collect Codeforces passwords. `cfx auth`
must continue to delegate authentication to a visible browser owned by the
user.
