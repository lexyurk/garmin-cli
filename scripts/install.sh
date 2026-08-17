#!/bin/sh

set -eu

REPOSITORY="lexyurk/garmin-cli"
GITHUB_URL="https://github.com"
LATEST_RELEASE_URL="$GITHUB_URL/$REPOSITORY/releases/latest"

temp_dir=""
install_temp=""

say() {
  printf '%s\n' "$*"
}

fail() {
  printf 'garmin-cli installer: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "$install_temp" ] && [ -f "$install_temp" ]; then
    rm -f "$install_temp"
  fi
  if [ -n "$temp_dir" ] && [ -d "$temp_dir" ]; then
    rm -rf "$temp_dir"
  fi
}

trap cleanup 0
trap 'exit 1' HUP INT TERM

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

map_os() {
  case "$1" in
    Linux) printf '%s\n' linux ;;
    *) fail "unsupported operating system: $1 (Linux is required)" ;;
  esac
}

map_arch() {
  case "$1" in
    x86_64|amd64) printf '%s\n' amd64 ;;
    arm64|aarch64) printf '%s\n' arm64 ;;
    *) fail "unsupported CPU architecture: $1 (x86_64/amd64 or arm64/aarch64 is required)" ;;
  esac
}

normalize_version() {
  requested=$1
  case "$requested" in
    v*) plain_version=${requested#v} ;;
    *) plain_version=$requested ;;
  esac

  case "$plain_version" in
    *.*.*) ;;
    *) fail "invalid GARMIN_VERSION: $requested (expected vMAJOR.MINOR.PATCH)" ;;
  esac
  major=${plain_version%%.*}
  remaining=${plain_version#*.}
  minor=${remaining%%.*}
  patch=${remaining#*.}
  case "$patch" in
    *.*) fail "invalid GARMIN_VERSION: $requested (expected vMAJOR.MINOR.PATCH)" ;;
  esac

  for component in "$major" "$minor" "$patch"; do
    case "$component" in
      ''|*[!0-9]*) fail "invalid GARMIN_VERSION: $requested (expected vMAJOR.MINOR.PATCH)" ;;
    esac
  done

  printf 'v%s\n' "$plain_version"
}

version_from_release_url() {
  release_url=$1
  release_prefix="$GITHUB_URL/$REPOSITORY/releases/tag/"
  case "$release_url" in
    "$release_prefix"*) release_version=${release_url#"$release_prefix"} ;;
    *) fail "GitHub returned an unexpected latest-release URL: $release_url" ;;
  esac
  normalize_version "$release_version"
}

resolve_version() {
  if [ -n "${GARMIN_VERSION:-}" ]; then
    normalize_version "$GARMIN_VERSION"
    return
  fi

  latest_url=$(curl --proto '=https' --tlsv1.2 -fsSL \
    -o /dev/null -w '%{url_effective}' "$LATEST_RELEASE_URL") || \
    fail "could not resolve the latest stable release"
  version_from_release_url "$latest_url"
}

release_download_url() {
  version=$1
  filename=$2
  printf '%s/%s/releases/download/%s/%s\n' "$GITHUB_URL" "$REPOSITORY" "$version" "$filename"
}

download() {
  url=$1
  destination=$2
  curl --proto '=https' --tlsv1.2 -fsSL --retry 3 --output "$destination" "$url" || \
    fail "download failed: $url"
}

verify_checksum() {
  archive=$1
  checksums=$2
  archive_name=$3

  expected=$(awk -v wanted="$archive_name" '
    $2 == wanted { count++; checksum = $1 }
    END {
      if (count != 1 || length(checksum) != 64 || checksum !~ /^[0-9A-Fa-f]+$/) exit 1
      print tolower(checksum)
    }
  ' "$checksums") || fail "checksums.txt has no unique valid SHA256 entry for $archive_name"

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$archive") || fail "could not hash $archive_name"
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$archive") || fail "could not hash $archive_name"
  else
    fail "SHA256 verification requires sha256sum or shasum"
  fi
  actual=${actual%% *}

  [ "$actual" = "$expected" ] || fail "SHA256 checksum mismatch for $archive_name"
}

install_binary() {
  source_binary=$1
  install_dir=$2

  if [ -e "$install_dir" ] && [ ! -d "$install_dir" ]; then
    fail "INSTALL_DIR is not a directory: $install_dir"
  fi
  mkdir -p "$install_dir" || fail "could not create INSTALL_DIR: $install_dir"
  [ -w "$install_dir" ] || fail "INSTALL_DIR is not writable: $install_dir"

  install_temp=$(mktemp "$install_dir/.garmin.XXXXXX") || \
    fail "could not create a temporary file in INSTALL_DIR: $install_dir"
  cp "$source_binary" "$install_temp" || fail "could not copy garmin into INSTALL_DIR"
  chmod 0755 "$install_temp" || fail "could not make garmin executable"
  mv -f "$install_temp" "$install_dir/garmin" || fail "could not install garmin into INSTALL_DIR"
  install_temp=""
}

warn_if_not_on_path() {
  install_dir=$1
  case ":${PATH:-}:" in
    *:"$install_dir":*) ;;
    *)
      printf 'Warning: %s is not on PATH. Add this to your shell profile:\n' "$install_dir" >&2
      printf '  export PATH="%s:$PATH"\n' "$install_dir" >&2
      ;;
  esac
}

main() {
  [ "$#" -eq 0 ] || fail "unexpected arguments; use GARMIN_VERSION and INSTALL_DIR environment variables"

  require_command uname
  require_command curl
  require_command tar
  require_command mktemp
  require_command awk
  require_command mkdir
  require_command cp
  require_command chmod
  require_command mv
  require_command rm
  if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
    fail "SHA256 verification requires sha256sum or shasum"
  fi

  os=$(map_os "$(uname -s)")
  arch=$(map_arch "$(uname -m)")
  version=$(resolve_version)
  plain_version=${version#v}

  if [ -n "${INSTALL_DIR:-}" ]; then
    install_dir=$INSTALL_DIR
  else
    [ -n "${HOME:-}" ] || fail "HOME is not set; set INSTALL_DIR to a user-writable directory"
    install_dir=$HOME/.local/bin
  fi

  temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/garmin-cli-install.XXXXXX") || \
    fail "could not create a temporary directory"

  archive_name="garmin-cli_${plain_version}_${os}_${arch}.tar.gz"
  archive="$temp_dir/$archive_name"
  checksums="$temp_dir/checksums.txt"

  say "Downloading garmin-cli $version for $os/$arch..."
  download "$(release_download_url "$version" "$archive_name")" "$archive"
  download "$(release_download_url "$version" checksums.txt)" "$checksums"
  verify_checksum "$archive" "$checksums" "$archive_name"

  tar -xzf "$archive" -C "$temp_dir" garmin || fail "could not extract garmin from $archive_name"
  [ -f "$temp_dir/garmin" ] || fail "release archive does not contain garmin"

  install_binary "$temp_dir/garmin" "$install_dir"
  say "Installed garmin-cli $version to $install_dir/garmin"
  warn_if_not_on_path "$install_dir"
}

if [ "${GARMIN_INSTALLER_TEST_MODE:-0}" != "1" ]; then
  main "$@"
fi
