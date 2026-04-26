#!/bin/sh
set -eu

repo="${SKIT_REPO:-vlln/skit}"
version="${SKIT_VERSION:-latest}"
install_dir="${SKIT_INSTALL_DIR:-$HOME/.local/bin}"
connect_timeout="${SKIT_CONNECT_TIMEOUT:-10}"
max_time="${SKIT_MAX_TIME:-300}"
speed_limit="${SKIT_SPEED_LIMIT:-1024}"
speed_time="${SKIT_SPEED_TIME:-30}"
retry="${SKIT_RETRY:-3}"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "skit installer: missing required command: $1" >&2
		exit 1
	fi
}

need curl
need tar

download() {
	url="$1"
	out="$2"
	label="$3"

	echo "skit installer: downloading $label" >&2
	if [ -t 2 ]; then
		curl -fL --progress-bar \
			--connect-timeout "$connect_timeout" \
			--max-time "$max_time" \
			--speed-limit "$speed_limit" \
			--speed-time "$speed_time" \
			--retry "$retry" \
			--retry-connrefused \
			"$url" -o "$out"
	else
		curl -fsSL \
			--connect-timeout "$connect_timeout" \
			--max-time "$max_time" \
			--speed-limit "$speed_limit" \
			--speed-time "$speed_time" \
			--retry "$retry" \
			--retry-connrefused \
			"$url" -o "$out"
	fi
}

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
base="${SKIT_DOWNLOAD_BASE:-$base}"

asset="skit_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "skit installer: installing $repo $version for $os/$arch" >&2
download "$base/checksums.txt" "$tmp/checksums.txt" "checksums.txt"
if ! grep "  $asset\$" "$tmp/checksums.txt" >/dev/null 2>&1; then
	echo "skit installer: checksums.txt does not list $asset" >&2
	exit 1
fi
download "$base/$asset" "$tmp/$asset" "$asset"
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
