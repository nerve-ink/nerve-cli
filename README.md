# nerve-cli

Small command-line sender for Nerve pipes.

`nerve send` reads plaintext from stdin, encrypts it locally with the sender key
from `NERVE_DSN`, and posts only ciphertext to the Nerve relay. The sender DSN
can send signals into one pipe, but it cannot read history or execute commands.

## Install

```bash
go install github.com/nerve-ink/nerve-cli/cmd/nerve@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Send

Copy a sender DSN from Nerve iOS, then:

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
