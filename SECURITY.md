# Security policy

## Supported versions

lazyfinder is pre-1.0 (`0.y.z`). Only the latest release receives security fixes.

| Version | Supported |
| ------- | --------- |
| latest  | yes       |

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Use GitHub's private vulnerability reporting: open the
[Security tab](https://github.com/MathiasDrizzy/lazyfinder/security) of this repository and choose
"Report a vulnerability". If that is not available to you, contact the maintainer through the
contact details on the [GitHub profile](https://github.com/MathiasDrizzy).

Include the lazyfinder version (`lazyfinder --version`), your operating system and terminal, and the steps to
reproduce. You can expect an acknowledgement within a few days. Once a fix is released, the report is credited
in the [changelog](CHANGELOG.md) unless you ask to stay anonymous.

## Scope

lazyfinder lists folders, previews files and starts programs you configured to open them. It never writes to
your files (milestone 1 is read-only). In scope: running anything that is not configured, shell injection through
file names, terminal escape sequence injection through file names or file content, decompression or image
"bombs" in the preview, and reading outside what the user navigated to.
