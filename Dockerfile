# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nerve ./cmd/nerve

FROM scratch

LABEL org.opencontainers.image.title="Nerve CLI"
LABEL org.opencontainers.image.description="Send failed deploy, CI, cron, backup, and smoke-test alerts to your phone with a send-only encrypted NerveOps secret."
LABEL org.opencontainers.image.source="https://github.com/nerve-ink/nerve-cli"
LABEL org.opencontainers.image.url="https://nerve.ink"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/nerve /nerve

USER 65532:65532

ENTRYPOINT ["/nerve"]
CMD ["--help"]
