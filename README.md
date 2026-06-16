# nerve-cli

[![Website](https://img.shields.io/badge/website-nerve.ink-111214)](https://nerve.ink)
[![Go Reference](https://pkg.go.dev/badge/github.com/nerve-ink/nerve-cli.svg)](https://pkg.go.dev/github.com/nerve-ink/nerve-cli)

[Website](https://nerve.ink) · [Start](https://nerve.ink/start) · [App Store](https://apps.apple.com/us/app/nerveops/id6778026992) · [Run agent](https://github.com/nerve-ink/nerve-agent)

Encrypted CI/CD, cron and server alerts to iPhone.

Sender secrets can only send. Relay sees ciphertext. Actions are optional.

```bash
echo "deploy failed" | nerve send
```

## Start with signals

1. Create a pipe in the Nerve mobile app.
2. Open **Pipe Setup** and choose **Send signals**.
3. Copy the sender DSN.
4. Install `nerve`.
5. Send your first encrypted alert:

```bash
curl -fsSL https://nerve.ink/install.sh | sh
export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"
echo "deploy failed" | nerve send --title "Nerve test"
```

Expected output:

```text
sent
```

The phone receives an encrypted signal. The relay routes ciphertext; decryption
happens locally on the device that owns the pipe.

NerveOps for iOS is available on the
[App Store](https://apps.apple.com/us/app/nerveops/id6778026992).

`nerve send` is the safe first Nerve integration. It reads plaintext from stdin,
encrypts it locally with the sender key from `NERVE_DSN`, and posts only
ciphertext to the Nerve relay.

The sender DSN can send signals into one pipe. It cannot read history, decrypt
old messages, connect as an agent, or execute commands.

## Why this is not just a webhook

| Boundary | What happens |
|---|---|
| CI secret leaks | A sender DSN can create alert noise in one pipe. It cannot read history, decrypt content, connect as an agent, or execute commands. |
| Relay receives traffic | The relay routes encrypted records and delivery metadata. Signal content is encrypted before it reaches the relay. |
| Phone receives push | APNs/FCM wake or notify the device. Clients sync the encrypted record and decrypt locally. |
| Agent is needed | Use [`nerve-agent`](https://github.com/nerve-ink/nerve-agent) only when you want signed actions on a machine you control. |

## Copy-paste Examples

- [GitHub Actions failure alert](./examples/github-actions-failure.yml)
- [Cron backup failure alert](./examples/cron-backup-failed.sh)
- [systemd failed unit alert](./examples/systemd-failed-unit.sh)

## Install

Recommended:

```bash
curl -fsSL https://nerve.ink/install.sh | sh
```

Go fallback / developer install:

```bash
go install github.com/nerve-ink/nerve-cli/cmd/nerve@latest
export PATH="$PATH:$(go env GOPATH)/bin"
nerve --help
```

## Docker

Use the Docker image when you want `nerve send` in CI, cron, or a host where you
do not want to install Go or a local binary.

```bash
echo "deploy failed" | docker run -i --rm \
  -e NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink" \
  p1xel32/nerve-cli:latest send
```

GitHub Container Registry mirror:

```bash
echo "deploy failed" | docker run -i --rm \
  -e NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink" \
  ghcr.io/nerve-ink/nerve-cli:latest send
```

The image is send-only. It does not run an agent and cannot execute commands.
For signed actions on a trusted host, use
[`nerve-agent`](https://github.com/nerve-ink/nerve-agent).

## Flags

```bash
echo "deploy failed" | nerve send \
  --severity critical \
  --title "Deploy failed"
```

Common flags:

| Flag | Default | Purpose |
|---|---:|---|
| `--dsn` | `NERVE_DSN` | Sender DSN. Prefer an env var or CI secret. |
| `--severity` | `standard` | Signal severity, for example `standard`, `alert`, `critical`. |
| `--title` | empty | Optional short title shown by clients. |
| `--kind` | `alert` | Signal kind metadata. |
| `--timeout` | `10s` | HTTP request timeout. |

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
      - name: Install nerve
        run: curl -fsSL https://nerve.ink/install.sh | sh
      - name: Send encrypted deploy signal
        env:
          NERVE_DSN: ${{ secrets.NERVE_DSN }}
        run: |
          printf 'deploy %s\nrun: %s\n' \
            "${{ github.event.workflow_run.conclusion }}" \
            "${{ github.event.workflow_run.html_url }}" \
            | nerve send --title "Backend Deploy"
```

For a regular job, send only on failure:

```yaml
- name: Install nerve
  run: curl -fsSL https://nerve.ink/install.sh | sh
- name: Notify Nerve on failure
  if: failure()
  env:
    NERVE_DSN: ${{ secrets.NERVE_DSN }}
  run: |
    printf 'FAILED: %s\nbranch: %s\nrun: %s\n' \
      "${{ github.repository }}" \
      "${{ github.ref_name }}" \
      "${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}" \
      | nerve send --severity critical --title "CI failed"
```

Keep the signal short: repository, branch/environment, status, commit SHA, and
run URL. Do not pipe full logs or secrets into phone alerts.

```text
Example message:
deploy success
sha: b62b038
run: https://github.com/nerve-ink/nerve-ops/actions/runs/26969431142
```

## Cron

```bash
#!/usr/bin/env bash
set -euo pipefail

export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"

if ! /opt/jobs/backup.sh; then
  echo "backup failed on $(hostname)" | nerve send --severity alert
  exit 1
fi
```

Crontab:

```cron
15 3 * * * /opt/jobs/backup-with-nerve.sh
```

## Bash / Smoke Tests

```bash
if ./smoke.sh; then
  echo "smoke passed: api-prod" | nerve send
else
  echo "smoke failed: api-prod" | nerve send --severity critical
fi
```

## Security Model

A sender DSN contains:

- a sender token
- a sender encryption key
- the relay host

If it leaks, an attacker can send fake signals into that one pipe until the
credential is rotated.

They cannot:

- read history
- decrypt old messages
- connect as an agent
- execute commands

Do not put secrets, tokens, private keys, or full environment dumps into signal
text. Send short operational context: service, environment, status, commit SHA,
and a run URL.

## Rotate a Sender DSN

1. Open pipe setup in the mobile app.
2. Create a new **Send signals** credential.
3. Update your CI secret or server env var.
4. Delete the old sender credential.

## Troubleshooting

### `go: command not found`

Use the recommended installer instead of the Go fallback:

```bash
curl -fsSL https://nerve.ink/install.sh | sh
```

The Go path is only for local development or environments that intentionally
build from source.

### `415 Unsupported Media Type`

You are probably calling the old `nerve` shell function or a generic webhook
instead of the Go CLI. Check:

```bash
type nerve
which nerve
nerve --help
```

The current CLI prints `sent` on success and sends
`Content-Type: application/json` automatically.

### `go install` cannot read the repository

Use the public module path exactly:

```bash
go install github.com/nerve-ink/nerve-cli/cmd/nerve@latest
```

If Go reports that GitHub needs credentials, verify that your environment is not
rewriting GitHub URLs to a private mirror or stale local fork.

### `nerve: NERVE_DSN or --dsn is required`

Export the DSN copied from pipe setup:

```bash
export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"
```

Or pass it directly:

```bash
echo "deploy failed" | nerve send --dsn "nerve://TOKEN:SENDER_KEY@api.nerve.ink"
```

### Signal says `sent`, but I do not see it

Check that:

- the phone is signed into the same Nerve account
- notifications are allowed
- the sender DSN belongs to the pipe you are watching
- the app can decrypt the pipe keys

## Agent

Need signed actions from a machine you control? Use
[`nerve-agent`](https://github.com/nerve-ink/nerve-agent).

The agent is the advanced path. Start with send-only signals unless you
explicitly need remote actions.
