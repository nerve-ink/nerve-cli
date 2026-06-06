# nerve-cli

[![Website](https://img.shields.io/badge/website-nerve.ink-111214)](https://nerve.ink)
[![Go Reference](https://pkg.go.dev/badge/github.com/nerve-ink/nerve-cli.svg)](https://pkg.go.dev/github.com/nerve-ink/nerve-cli)

[Website](https://nerve.ink) · [Docs](https://nerve.ink/docs.html) · [Run agent](https://github.com/nerve-ink/nerve-agent)

Small command-line sender for Nerve pipes. This is the recommended first
integration path because it is one-way and does not grant shell access.

`nerve send` reads plaintext from stdin, encrypts it locally with the sender key
from `NERVE_DSN`, and posts only ciphertext to the Nerve relay. The sender DSN
can send signals into one pipe, but it cannot read history or execute commands.

## Install

```bash
go install github.com/nerve-ink/nerve-cli/cmd/nerve@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Send

Create a pipe in Nerve iOS, open Pipe Setup, choose **Send signals**, then copy
the sender DSN:

```bash
export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"
echo "deploy failed" | nerve send
```

Optional flags:

```bash
echo "deploy failed" | nerve send --severity critical --title "Deploy failed"
```

The relay receives an encrypted `sender_v1` payload. Decryption happens on the
iOS device that owns the pipe.

## GitHub Actions

Store the sender DSN as a repository or organization secret named `NERVE_DSN`.

```yaml
name: Notify Nerve

on:
  workflow_run:
    workflows: ["Backend Deploy"]
    types: [completed]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
      - run: go install github.com/nerve-ink/nerve-cli/cmd/nerve@latest
      - name: Send encrypted deploy signal
        env:
          NERVE_DSN: ${{ secrets.NERVE_DSN }}
        run: |
          printf 'deploy %s\nrun: %s\n' \
            "${{ github.event.workflow_run.conclusion }}" \
            "${{ github.event.workflow_run.html_url }}" | nerve send --title "Backend Deploy"
```

## Security Note

A sender DSN contains a sender token and a sender encryption key.

If it leaks, an attacker can send fake signals into that one pipe until the
credential is rotated. They cannot read history, decrypt old messages, connect
as an agent, or execute commands.
