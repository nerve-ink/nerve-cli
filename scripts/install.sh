#!/usr/bin/env sh
set -eu

REPO="${NERVE_CLI_REPO:-nerve-ink/nerve-cli}"
VERSION="${NERVE_CLI_VERSION:-latest}"

fail() {
  echo "nerve install: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

need uname
need curl

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux|darwin) ;;
  *) fail "unsupported OS: $os" ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $arch" ;;
esac

asset="nerve-${os}-${arch}"
base="https://github.com/${REPO}/releases"
if [ "$VERSION" = "latest" ]; then
  download_base="${base}/latest/download"
else
  download_base="${base}/download/${VERSION}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT HUP TERM

echo "Downloading ${asset} from ${REPO} ${VERSION}..."
curl -fsSL "${download_base}/${asset}" -o "${tmp_dir}/nerve"
chmod +x "${tmp_dir}/nerve"

if curl -fsSL "${download_base}/checksums.txt" -o "${tmp_dir}/checksums.txt"; then
  if command -v shasum >/dev/null 2>&1; then
    expected="$(grep " ${asset}\$" "${tmp_dir}/checksums.txt" | awk '{print $1}')"
    [ -n "$expected" ] || fail "checksum for ${asset} not found"
    actual="$(shasum -a 256 "${tmp_dir}/nerve" | awk '{print $1}')"
    [ "$expected" = "$actual" ] || fail "checksum mismatch"
  else
    echo "shasum not found; skipping checksum verification" >&2
  fi
else
  echo "checksums.txt unavailable; skipping checksum verification" >&2
fi

install_dir="/usr/local/bin"
if [ -w "$install_dir" ]; then
  cp "${tmp_dir}/nerve" "${install_dir}/nerve"
else
  install_dir="${HOME}/.local/bin"
  mkdir -p "$install_dir"
  cp "${tmp_dir}/nerve" "${install_dir}/nerve"
fi

echo "Installed nerve to ${install_dir}/nerve"
if command -v "${install_dir}/nerve" >/dev/null 2>&1; then
  echo "Version: $("${install_dir}/nerve" --version)"
fi

case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *) echo "Add ${install_dir} to PATH if 'nerve' is not found." ;;
esac
