#!/bin/sh
set -eu

repo="${SKIT_REPO:-vlln/skit}"
version="${SKIT_VERSION:-latest}"
install_dir="${SKIT_INSTALL_DIR:-$HOME/.local/bin}"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "skit installer: missing required command: $1" >&2
		exit 1
	fi
}

need curl
need tar

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
Darwin) os="Darwin" ;;
Linux) os="Linux" ;;
*) echo "skit installer: unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
x86_64 | amd64) arch="x86_64" ;;
arm64 | aarch64) arch="arm64" ;;
*) echo "skit installer: unsupported architecture: $arch" >&2; exit 1 ;;
esac

if [ "$version" = "latest" ]; then
	base="https://github.com/$repo/releases/latest/download"
else
	base="https://github.com/$repo/releases/download/$version"
fi

asset="skit_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"
(
	cd "$tmp"
	if command -v sha256sum >/dev/null 2>&1; then
		grep "  $asset\$" checksums.txt | sha256sum -c -
	elif command -v shasum >/dev/null 2>&1; then
		grep "  $asset\$" checksums.txt | shasum -a 256 -c -
	else
		echo "skit installer: sha256sum or shasum is required for checksum verification" >&2
		exit 1
	fi
)

tar -xzf "$tmp/$asset" -C "$tmp"
if [ ! -f "$tmp/skit" ]; then
	echo "skit installer: release archive did not contain skit" >&2
	exit 1
fi

mkdir -p "$install_dir"
install -m 0755 "$tmp/skit" "$install_dir/skit"

echo "installed skit to $install_dir/skit"
case ":$PATH:" in
*":$install_dir:"*) ;;
*) echo "add $install_dir to PATH to run skit directly" ;;
esac
