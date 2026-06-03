# Security Policy

## Supported Versions

We do not maintain long-term support for individual releases. Security fixes are included in the next published version only; we do not backport fixes to older releases.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| Older   | :x:                |

Check [Releases](https://github.com/webp-sh/webp_server_go/releases) for the current version and upgrade when a security fix is published.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you believe you have found a security issue in WebP Server Go, report it privately so we can investigate and release a fix before details are disclosed publicly.

### Preferred: GitHub Private Security Advisory

Use [GitHub Security Advisories](https://github.com/webp-sh/webp_server_go/security/advisories/new) to submit a private report. This is the fastest way for maintainers to receive, track, and coordinate a fix.

### What to Include

- A clear description of the vulnerability and its potential impact
- Steps to reproduce, including request examples, configuration, or proof-of-concept if available
- Affected version(s) or commit hash
- Any suggested mitigation or fix, if you have one

### Response Timeline

We aim to:

- Acknowledge your report within **3 business days**
- Provide an initial assessment within **7 business days**
- Keep you informed as we work on a fix and coordinated disclosure

Timelines may vary for complex issues or during holidays; we will communicate delays when they occur.

### Disclosure

We follow coordinated disclosure. Please allow reasonable time for a fix before public disclosure. We will credit reporters in the advisory when they wish to be acknowledged.

## Scope

The following are generally **in scope**:

- Remote code execution, authentication bypass, or privilege escalation in WebP Server Go
- Path traversal or unauthorized file access via HTTP requests
- Server-side request forgery (SSRF) or unsafe remote fetching in proxy/remote modes
- Denial of service that can be triggered remotely with minimal resources
- Memory safety or parser issues in image processing that lead to exploitable conditions

The following are generally **out of scope**:

- Issues in third-party dependencies already tracked upstream (please still report if the project is affected and we have not patched)
- Vulnerabilities in your reverse proxy, container runtime, or host OS configuration
- Missing security headers or TLS configuration on your deployment (see deployment guidance below)
- Social engineering or physical access attacks

## Deployment Guidance

WebP Server Go is an HTTP image conversion service. Secure deployment is shared responsibility between the software and operators.

We recommend:

- **Bind locally by default** — keep `HOST` at `127.0.0.1` and expose the service through a reverse proxy (Nginx, Caddy, etc.), as shown in the [README](./README.md).
- **Use Docker** — our maintained images are scanned in CI; upgrade to the latest release when security fixes are published.
- **Limit exposed surface** — restrict `ALLOWED_TYPES`, disable unused features (`ENABLE_EXTRA_PARAMS`, remote/proxy modes) when not required.
- **Protect origin paths** — ensure `IMG_PATH` and cache directories are not writable by untrusted users.
- **Review `IMG_MAP` / remote sources** — proxy and remote modes fetch URLs; only map to trusted origins to reduce SSRF risk.
- **Keep dependencies current** — rebuild or pull new images after security releases.

## Security Practices in This Repository

- Static analysis with [CodeQL](https://github.com/webp-sh/webp_server_go/actions/workflows/codeql-analysis.yml)
- Container image scanning with Trivy in [CI](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml)
- Dependency updates via [Dependabot](https://github.com/webp-sh/webp_server_go/blob/master/.github/dependabot.yml)

Thank you for helping keep WebP Server Go and its users safe.
