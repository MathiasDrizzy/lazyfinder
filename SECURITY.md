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

## Verifying a release

Every release archive has an SPDX SBOM next to it (a `.sbom.json` file made by Syft), and a signed build provenance
attestation ties each archive and each SBOM to the commit and the workflow that built them. With the GitHub CLI and
nothing else installed:

```sh
gh attestation verify lazyfinder_<version>_<os>_<arch>.tar.gz -R MathiasDrizzy/lazyfinder
```

It exits with 0 when the file is the one the release workflow built, and fails if the file was changed. The same
command works for the `.sbom.json` files. `checksums.txt` is an index and has no attestation of its own; the checksum
of a download is still checked with `shasum -a 256 -c checksums.txt --ignore-missing`.

## Scope

lazyfinder lists folders, previews files, runs the file operations you ask for (copy, move, rename, move to the
Trash, undo) and starts the programs you configured to open files. Your own `config.json` is trusted; the folders you
look at are not. In scope:

- running anything that is not configured, or shell or argument injection through file names;
- terminal escape sequence injection through file names, archive entries or file content;
- a git repository from somewhere else running a program from its own `.git/config` (fsmonitor, filters, textconv)
  while lazyfinder only reads it. A git operation you ask for (stage, discard, stash, switch, fetch, pull, push) runs
  the repository's config as `git` does in a terminal, and is out of scope ([ADR 0019](docs/adr/0019-git-writes-trust-the-repository.md));
- a FIFO, a device or a huge file in a folder blocking or exhausting the app, and decompression or image "bombs";
- a file operation that deletes for good, overwrites, or writes outside the folder you chose.
