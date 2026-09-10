#!/bin/sh
set -e

REPO="sai-toolboxs/noti-cli"
BINARY="noti-cli"

# Detect OS
detect_os() {
  os=$(uname -s)
  case "$os" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)
      echo "error: unsupported OS: $os" >&2
      exit 1
      ;;
  esac
}

# Detect architecture
detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64)   echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    armv7l|armhf)
      if [ -n "$TERMUX_VERSION" ] || [ -d "/data/data/com.termux" ]; then
        echo "arm64"
      else
        echo "error: unsupported architecture: $arch (use Go install instead)" >&2
        exit 1
      fi
      ;;
    *)
      echo "error: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

# Detect install directory
detect_install_dir() {
  if [ -n "$TERMUX_VERSION" ] || [ -d "/data/data/com.termux" ]; then
    echo "$PREFIX/bin"
  else
    echo "$HOME/.local/bin"
  fi
}

# Get latest release version from GitHub API
get_latest_version() {
  url="https://api.github.com/repos/${REPO}/releases/latest"
  if command -v curl >/dev/null 2>&1; then
    version=$(curl -fsSL "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
  elif command -v wget >/dev/null 2>&1; then
    version=$(wget -qO- "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
  else
    echo "error: curl or wget required" >&2
    exit 1
  fi

  if [ -z "$version" ]; then
    echo "error: could not determine latest version" >&2
    exit 1
  fi

  echo "$version"
}

# Verify SHA-256 checksum
verify_checksum() {
  file="$1"
  checksum_file="$2"

  if command -v sha256sum >/dev/null 2>&1; then
    expected=$(grep "$(basename "$file")" "$checksum_file" | awk '{print $1}')
    actual=$(sha256sum "$file" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    expected=$(grep "$(basename "$file")" "$checksum_file" | awk '{print $1}')
    actual=$(shasum -a 256 "$file" | awk '{print $1}')
  else
    echo "warning: no sha256sum or shasum found, skipping checksum verification" >&2
    return 0
  fi

  if [ -z "$expected" ]; then
    echo "error: checksum not found for $(basename "$file")" >&2
    exit 1
  fi

  if [ "$expected" != "$actual" ]; then
    echo "error: checksum mismatch" >&2
    echo "  expected: $expected" >&2
    echo "  actual:   $actual" >&2
    exit 1
  fi

  echo "checksum verified"
}

# Main
main() {
  OS=$(detect_os)
  ARCH=$(detect_arch)
  INSTALL_DIR=$(detect_install_dir)
  VERSION=$(get_latest_version)
  VERSION_NUM=${VERSION#v}

  ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
  CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

  echo "noti-cli installer"
  echo "  version:    ${VERSION}"
  echo "  os:         ${OS}"
  echo "  arch:       ${ARCH}"
  echo "  install to: ${INSTALL_DIR}"
  echo ""

  # Create install directory
  mkdir -p "$INSTALL_DIR"

  # Download archive and checksums
  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  echo "downloading ${ARCHIVE}..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL"
    curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${TMPDIR}/${ARCHIVE}" "$URL"
    wget -qO "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"
  else
    echo "error: curl or wget required" >&2
    exit 1
  fi

  # Verify checksum
  echo "verifying checksum..."
  verify_checksum "${TMPDIR}/${ARCHIVE}" "${TMPDIR}/checksums.txt"

  # Extract binary
  echo "extracting..."
  tar xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

  # Install
  if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    echo "error: cannot write to ${INSTALL_DIR}" >&2
    echo "try: sudo mv ${TMPDIR}/${BINARY} ${INSTALL_DIR}/${BINARY}" >&2
    exit 1
  fi

  chmod +x "${INSTALL_DIR}/${BINARY}"

  echo ""
  echo "installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
  echo ""
  echo "add ${INSTALL_DIR} to your PATH if not already present:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
}

main "$@"
