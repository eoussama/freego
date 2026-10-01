# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✓         |
| < 0.1   | ✗ (built for the deprecated FreeStuff API v1) |

## Reporting a Vulnerability

Please report vulnerabilities privately through [GitHub's private vulnerability reporting](https://github.com/eoussama/freego/security/advisories/new), so they can be fixed before being disclosed. Include a clear description of the vulnerability and the steps to reproduce it.

For issues that are not sensitive, you can use the [Issues](https://github.com/eoussama/freego/issues) tab.

## Handling Credentials

- Never commit your `.env` file; it is listed in `.gitignore` and `.dockerignore`.
- Your FreeStuff REST API key is a secret. The webhook public key is not, but verifying deliveries with it is what protects your webhook endpoint from forged requests.
