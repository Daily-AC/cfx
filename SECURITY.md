# Security policy

## Reporting

Please report vulnerabilities through GitHub's private security advisory
feature. Do not open a public issue for vulnerabilities involving session
cookies, arbitrary command execution, or unintended submissions.

## Local trust model

Dedicated mode launches a Chrome profile in the operating system's user config
directory. That profile contains Codeforces session cookies and should be
treated as sensitive. Extension mode reuses the Chrome profile selected by the
user and does not copy its cookies. `cfx` never reads passwords or prints
cookies.

Training workspaces can execute user-edited C++ or Python code through
`cfx test`. Only run workspaces you trust. Submission is an explicit external
action and requires `cfx submit --yes`.

No OpenCLI service, hosted relay, analytics endpoint, or telemetry service is
used. The optional embedded extension talks only to Codeforces and a temporary
localhost bridge. The bridge requires a per-installation random token and is
bound to `127.0.0.1`; the token and unpacked extension are stored under the user
config directory. Requests with an `Origin` header are accepted only from a
Chrome extension origin; Chrome may omit that header for host-permission
requests, which still require the token. Codeforces API requests and browser
navigation go directly from the user's machine to Codeforces.
