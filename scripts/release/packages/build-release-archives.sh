#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

version=""
out_dir=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/packages/build-release-archives.sh [--version vX.Y.Z] [--out DIR]

Builds the installable linux-x64 release archive:
- tetra-<version>-linux-x64.tar.gz
- tetra-<version>-linux-x64.sha256
- install.sh
- checksums.txt
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      shift
      if [[ $# -eq 0 ]]; then
        echo "build-release-archives: --version requires a value" >&2
        exit 2
      fi
      version="$1"
      ;;
    --out)
      shift
      if [[ $# -eq 0 ]]; then
        echo "build-release-archives: --out requires a value" >&2
        exit 2
      fi
      out_dir="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "build-release-archives: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ -z "$version" ]]; then
  version="$(sed -n 's/^const CompilerVersion = "\(.*\)"$/\1/p' compiler/internal/version/version.go)"
fi
if [[ -z "$version" ]]; then
  echo "build-release-archives: could not determine compiler version" >&2
  exit 1
fi
case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *)
    echo "build-release-archives: version must look like vX.Y.Z, got $version" >&2
    exit 2
    ;;
esac

if [[ -z "$out_dir" ]]; then
  out_dir="dist/releases/$version"
fi

target="linux-x64"
archive_name="tetra-${version}-${target}.tar.gz"
source_archive_name="tetra-${version}-source.tar.gz"
package_root_name="tetra-${version}-${target}"
build_root="$out_dir/.build"
package_root="$build_root/$package_root_name"

rm -rf "$build_root"
mkdir -p "$package_root/bin" "$out_dir"
install -m 0755 install.sh "$out_dir/install.sh"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-release-packages"
fi
mkdir -p "$GOCACHE"

echo "Building Tetra $version for $target"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$package_root/bin/tetra" ./cli/cmd/tetra
cp "$package_root/bin/tetra" "$package_root/bin/t"

cp README.md "$package_root/README.md"
if [[ -f LICENSE ]]; then
  cp LICENSE "$package_root/LICENSE"
fi

cat > "$package_root/MANIFEST.txt" <<EOF
Tetra Language $version
Target: $target
Contents:
- bin/tetra
- bin/t
- README.md
- LICENSE
EOF

(
  cd "$build_root"
  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="UTC 2026-06-06" -czf "../$archive_name" "$package_root_name"
)

git archive --format=tar.gz --prefix="tetra-${version}-source/" -o "$out_dir/$source_archive_name" HEAD

(
  cd "$out_dir"
  sha256sum "$archive_name" > "${archive_name}.sha256"
  sha256sum "$source_archive_name" > "${source_archive_name}.sha256"
  sha256sum "$archive_name" "$source_archive_name" > checksums.txt
)

echo "Archive: $out_dir/$archive_name"
echo "Source: $out_dir/$source_archive_name"
echo "Installer: $out_dir/install.sh"
echo "Checksum: $out_dir/checksums.txt"
