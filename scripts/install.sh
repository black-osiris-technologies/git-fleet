#!/bin/sh
# Install git-fleet on Linux or macOS by downloading the release binary for the
# current OS and architecture. No Go toolchain is required.
#
#   curl -fsSL https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.sh | sh
#
# Environment variables:
#   GIT_FLEET_VERSION      version tag to install (default: latest release)
#   GIT_FLEET_INSTALL_DIR  install directory (default: /usr/local/bin, else ~/.local/bin)
set -eu

REPO="black-osiris-technologies/git-fleet"
BINARY="git-fleet"
INSTALL_DIR="${GIT_FLEET_INSTALL_DIR:-/usr/local/bin}"

fail() {
	echo "install: $1" >&2
	exit 1
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
	linux | darwin) ;;
	*) fail "unsupported OS '$os'; on Windows use scripts/install.ps1" ;;
esac

arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	*) fail "unsupported architecture '$arch'" ;;
esac

download() {
	# download <url> <output>
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

version="${GIT_FLEET_VERSION:-}"
if [ -z "$version" ]; then
	api="https://api.github.com/repos/$REPO/releases/latest"
	version=$(download "$api" /dev/stdout | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')
fi
[ -n "$version" ] || fail "could not determine the latest version"

number="${version#v}"
archive="${BINARY}_${number}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$version/$archive"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $BINARY $version ($os/$arch)"
download "$url" "$tmp/$archive"
tar -xzf "$tmp/$archive" -C "$tmp"
[ -f "$tmp/$BINARY" ] || fail "archive did not contain $BINARY"

install_binary() {
	# install_binary <dir>
	mkdir -p "$1"
	install -m 0755 "$tmp/$BINARY" "$1/$BINARY"
}

if [ -w "$INSTALL_DIR" ] || { [ ! -e "$INSTALL_DIR" ] && mkdir -p "$INSTALL_DIR" 2>/dev/null; }; then
	install_binary "$INSTALL_DIR"
elif [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1; then
	echo "Installing to $INSTALL_DIR (requires sudo)"
	sudo install -d -m 0755 "$INSTALL_DIR"
	sudo install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
else
	INSTALL_DIR="$HOME/.local/bin"
	install_binary "$INSTALL_DIR"
	case ":$PATH:" in
		*":$INSTALL_DIR:"*) ;;
		*) echo "Note: $INSTALL_DIR is not on your PATH; add it to your shell profile." ;;
	esac
fi

echo "Installed $BINARY $version to $INSTALL_DIR"
