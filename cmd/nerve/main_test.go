package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseSenderDSN(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytesOf(32, 7))
	dsn, err := parseSenderDSN("nerve://tok_123:" + key + "@api.nerve.ink")
	if err != nil {
		t.Fatalf("parseSenderDSN returned error: %v", err)
	}
	if dsn.Token != "tok_123" {
		t.Fatalf("token = %q", dsn.Token)
	}
	if dsn.Host != "api.nerve.ink" {
		t.Fatalf("host = %q", dsn.Host)
	}
	if dsn.Scheme != "https" {
		t.Fatalf("scheme = %q", dsn.Scheme)
	}
	if len(dsn.Key) != 32 {
		t.Fatalf("key len = %d", len(dsn.Key))
	}
}

func TestEncryptSenderPayloadRoundTrip(t *testing.T) {
	key := bytesOf(32, 3)
	payload, err := encryptSenderPayload(key, []byte("deploy failed"))
	if err != nil {
		t.Fatalf("encryptSenderPayload returned error: %v", err)
	}

	packed, err := base64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("AES cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("GCM: %v", err)
	}
	if len(packed) <= gcm.NonceSize() {
		t.Fatalf("packed payload too short: %d", len(packed))
	}
	nonce := packed[:gcm.NonceSize()]
	ciphertext := packed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(senderAAD))
	if err != nil {
		t.Fatalf("decrypt payload: %v", err)
	}
	if string(plaintext) != "deploy failed" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}

func TestRunSendPostsSenderV1Payload(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytesOf(32, 9))
	var sawRequest bool

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawRequest = true
		if r.URL.Path != "/api/v2/hooks/tok_123" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Scheme != "https" || r.URL.Host != "api.nerve.ink" {
			t.Fatalf("url = %s", r.URL.String())
		}
		if got := r.Header.Get("X-Nerve-Encryption-Mode"); got != "sender_v1" {
			t.Fatalf("encryption header = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q", got)
		}
		if got := r.Header.Get("X-Nerve-Severity"); got != "critical" {
			t.Fatalf("severity header = %q", got)
		}
		if got := r.Header.Get("X-Nerve-Title"); got != "Deploy failed" {
			t.Fatalf("title header = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if len(body) == 0 || strings.Contains(string(body), "secret plaintext") {
			t.Fatalf("body was not encrypted: %q", body)
		}
		var requestBody struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			t.Fatalf("body was not json: %v", err)
		}
		if requestBody.Text == "" {
			t.Fatal("encrypted text is empty")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	dsn := "nerve://tok_123:" + key + "@api.nerve.ink"
	var out strings.Builder

	err := run(
		[]string{"send", "--dsn", dsn, "--severity", "critical", "--title", "Deploy failed"},
		strings.NewReader("secret plaintext\n"),
		&out,
		client,
	)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !sawRequest {
		t.Fatal("server did not receive request")
	}
	if strings.TrimSpace(out.String()) != "sent" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunSendFailsOnNon2xx(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytesOf(32, 9))
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(strings.NewReader(`{"title":"Not Found","detail":"Invalid Token"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	dsn := "nerve://tok_123:" + key + "@api.nerve.ink"
	var out strings.Builder
	err := run([]string{"send", "--dsn", dsn}, strings.NewReader("deploy failed\n"), &out, client)
	if err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
	if !strings.Contains(err.Error(), "404 Not Found") || !strings.Contains(err.Error(), "Invalid Token") {
		t.Fatalf("error = %q", err.Error())
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunHelpExitsCleanly(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"help"},
		{"send", "--help"},
		{"send", "-h"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out strings.Builder
			err := run(args, strings.NewReader(""), &out, nil)
			if err != nil {
				t.Fatalf("run returned error: %v", err)
			}
			if !strings.Contains(out.String(), "Usage:") || !strings.Contains(out.String(), "nerve send") {
				t.Fatalf("stdout = %q", out.String())
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func bytesOf(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
