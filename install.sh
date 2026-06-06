#!/usr/bin/env bash
set -euo pipefail

repo="${TETRA_REPO:-BoSuY0/Tetra_Language}"
version="${TETRA_VERSION:-v0.4.0}"
install_dir="${TETRA_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
  cat <<'USAGE'
Usage: curl -fsSL https://raw.githubusercontent.com/BoSuY0/Tetra_Language/main/install.sh | bash

Environment:
  TETRA_VERSION      Release version to install. Default: v0.4.0
  TETRA_REPO         GitHub repository. Default: BoSuY0/Tetra_Language
  TETRA_INSTALL_DIR  Install directory. Default: $HOME/.local/bin
  TETRA_BASE_URL     Override release download base URL.
  TETRA_ARCHIVE_URL  Override archive URL directly.
  TETRA_CHECKSUM_URL Override checksums.txt URL directly.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

case "$(uname -s)" in
  Linux) os="linux" ;;
  *)
    echo "install.sh: only Linux x64 release assets are published for this channel" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="x64" ;;
  *)
    echo "install.sh: only Linux x64 release assets are published for this channel" >&2
    exit 1
    ;;
esac

target="${os}-${arch}"
asset="tetra-${version}-${target}.tar.gz"
base_url="${TETRA_BASE_URL:-https://github.com/${repo}/releases/download/${version}}"
archive_url="${TETRA_ARCHIVE_URL:-${base_url}/${asset}}"
checksum_url="${TETRA_CHECKSUM_URL:-${base_url}/checksums.txt}"

if command -v curl >/dev/null 2>&1; then
  download() {
    curl -fsSL "$1" -o "$2"
  }
elif command -v wget >/dev/null 2>&1; then
  download() {
    wget -qO "$2" "$1"
  }
else
  echo "install.sh: curl or wget is required" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

echo "Downloading $archive_url"
download "$archive_url" "$tmp_dir/$asset"
download "$checksum_url" "$tmp_dir/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  expected_line="$(grep -F " $asset" "$tmp_dir/checksums.txt" || true)"
  if [[ -z "$expected_line" ]]; then
    echo "install.sh: checksums.txt does not contain $asset" >&2
    exit 1
  fi
  (
    cd "$tmp_dir"
    printf '%s\n' "$expected_line" | sha256sum -c -
  )
else
  echo "install.sh: sha256sum is required for checksum verification" >&2
  exit 1
fi

mkdir -p "$tmp_dir/unpack"
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir/unpack"

tetra_bin="$(find "$tmp_dir/unpack" -type f -path '*/bin/tetra' -print -quit)"
t_bin="$(find "$tmp_dir/unpack" -type f -path '*/bin/t' -print -quit)"
if [[ -z "$tetra_bin" || -z "$t_bin" ]]; then
  echo "install.sh: archive did not contain bin/tetra and bin/t" >&2
  exit 1
fi

mkdir -p "$install_dir"
install -m 0755 "$tetra_bin" "$install_dir/tetra"
install -m 0755 "$t_bin" "$install_dir/t"

echo "Installed Tetra $version to $install_dir"
if ! command -v tetra >/dev/null 2>&1 && [[ ":$PATH:" != *":$install_dir:"* ]]; then
  echo "Add this to your shell profile if needed:"
  echo "  export PATH=\"$install_dir:\$PATH\""
fi
"$install_dir/tetra" version
