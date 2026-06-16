# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nerve ./cmd/nerve

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/nerve /nerve

USER 65532:65532

ENTRYPOINT ["/nerve"]
CMD ["--help"]
