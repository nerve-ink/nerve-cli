package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
	senderAAD      = "nerve.sender.v1"
)

type senderDSN struct {
	Token  string
	Key    []byte
	Host   string
	Scheme string
}

type sendConfig struct {
	DSN      string
	Title    string
	Severity string
	Kind     string
	Timeout  time.Duration
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, http.DefaultClient); err != nil {
		fmt.Fprintf(os.Stderr, "nerve: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, client *http.Client) error {
	if len(args) > 0 && args[0] == "send" {
		args = args[1:]
	}

	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	if cfg.DSN == "" {
		cfg.DSN = os.Getenv("NERVE_DSN")
	}
	if cfg.DSN == "" {
		return errors.New("NERVE_DSN or --dsn is required")
	}

	plaintext, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	plaintext = bytes.TrimRight(plaintext, "\r\n")
	if len(plaintext) == 0 {
		return errors.New("stdin is empty")
	}

	dsn, err := parseSenderDSN(cfg.DSN)
	if err != nil {
		return err
	}

	payload, err := encryptSenderPayload(dsn.Key, plaintext)
	if err != nil {
		return err
	}

	if err := postSenderPayload(client, cfg, dsn, payload); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "sent")
	return nil
}

func parseFlags(args []string) (sendConfig, error) {
	fs := flag.NewFlagSet("nerve send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cfg sendConfig
	fs.StringVar(&cfg.DSN, "dsn", "", "sender DSN; defaults to NERVE_DSN")
	fs.StringVar(&cfg.Title, "title", "", "signal title")
	fs.StringVar(&cfg.Severity, "severity", "standard", "signal severity")
	fs.StringVar(&cfg.Kind, "kind", "alert", "signal kind")
	fs.DurationVar(&cfg.Timeout, "timeout", defaultTimeout, "HTTP request timeout")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected argument %q; pipe data on stdin", fs.Arg(0))
	}
	return cfg, nil
}

func parseSenderDSN(raw string) (senderDSN, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return senderDSN{}, fmt.Errorf("parse DSN: %w", err)
	}
	if parsed.Scheme != "nerve" {
		return senderDSN{}, errors.New("DSN scheme must be nerve")
	}
	token := parsed.User.Username()
	keyText, hasKey := parsed.User.Password()
	if token == "" || !hasKey || keyText == "" {
		return senderDSN{}, errors.New("DSN must include token and sender key")
	}
	if parsed.Host == "" {
		return senderDSN{}, errors.New("DSN must include relay host")
	}

	key, err := decodeSenderKey(keyText)
	if err != nil {
		return senderDSN{}, err
	}

	scheme := parsed.Query().Get("scheme")
	if scheme == "" {
		scheme = "https"
		if strings.HasPrefix(parsed.Host, "localhost") ||
			strings.HasPrefix(parsed.Host, "127.0.0.1") ||
			strings.HasPrefix(parsed.Host, "[::1]") {
			scheme = "http"
		}
	}
	if scheme != "http" && scheme != "https" {
		return senderDSN{}, errors.New("DSN scheme query must be http or https")
	}

	return senderDSN{
		Token:  token,
		Key:    key,
		Host:   parsed.Host,
		Scheme: scheme,
	}, nil
}

func decodeSenderKey(text string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, enc := range encodings {
		decoded, err := enc.DecodeString(text)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	return nil, errors.New("sender key must be 32 bytes encoded as base64")
}

func encryptSenderPayload(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte(senderAAD))
	packed := make([]byte, 0, len(nonce)+len(ciphertext))
	packed = append(packed, nonce...)
	packed = append(packed, ciphertext...)

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(packed)))
	base64.StdEncoding.Encode(encoded, packed)
	return encoded, nil
}

func postSenderPayload(client *http.Client, cfg sendConfig, dsn senderDSN, payload []byte) error {
	if client == nil {
		client = http.DefaultClient
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	endpoint := fmt.Sprintf("%s://%s/api/v2/hooks/%s", dsn.Scheme, dsn.Host, url.PathEscape(dsn.Token))
	body, err := json.Marshal(map[string]string{"text": string(payload)})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nerve-Encryption-Mode", "sender_v1")
	if cfg.Severity != "" {
		req.Header.Set("X-Nerve-Severity", cfg.Severity)
	}
	if cfg.Kind != "" {
		req.Header.Set("X-Nerve-Kind", cfg.Kind)
	}
	if cfg.Title != "" {
		req.Header.Set("X-Nerve-Title", cfg.Title)
	}

	httpClient := *client
	httpClient.Timeout = timeout
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send signal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("send signal: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
