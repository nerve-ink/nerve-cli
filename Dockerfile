# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nerve ./cmd/nerve

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
  && addgroup -S nerve \
  && adduser -S -G nerve nerve

COPY --from=build /out/nerve /usr/local/bin/nerve

USER nerve
WORKDIR /work

ENTRYPOINT ["nerve"]
CMD ["--help"]
