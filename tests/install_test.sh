#!/bin/sh

set -eu

repo_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
GARMIN_INSTALLER_TEST_MODE=1
export GARMIN_INSTALLER_TEST_MODE
. "$repo_dir/scripts/install.sh"
trap - 0 HUP INT TERM

tests_run=0

assert_eq() {
  expected_value=$1
  actual_value=$2
  description=$3
  tests_run=$((tests_run + 1))
  if [ "$expected_value" != "$actual_value" ]; then
    printf 'not ok %s: expected <%s>, got <%s>\n' "$description" "$expected_value" "$actual_value" >&2
    exit 1
  fi
  printf 'ok %s\n' "$description"
}

assert_fails() {
  description=$1
  shift
  tests_run=$((tests_run + 1))
  if ( "$@" ) >/dev/null 2>&1; then
    printf 'not ok %s: command unexpectedly succeeded\n' "$description" >&2
    exit 1
  fi
  printf 'ok %s\n' "$description"
}

assert_eq linux "$(map_os Linux)" "maps Linux"
assert_eq amd64 "$(map_arch x86_64)" "maps x86_64"
assert_eq amd64 "$(map_arch amd64)" "maps amd64"
assert_eq arm64 "$(map_arch arm64)" "maps arm64"
assert_eq arm64 "$(map_arch aarch64)" "maps aarch64"
assert_fails "rejects unsupported OS" map_os Darwin
assert_fails "rejects unsupported architecture" map_arch riscv64
assert_fails "rejects unexpected arguments" main --version v0.1.0

assert_eq v0.1.0 "$(normalize_version v0.1.0)" "keeps a prefixed version"
assert_eq v1.2.3 "$(normalize_version 1.2.3)" "adds a version prefix"
assert_fails "rejects prerelease version override" normalize_version v1.2.3-rc1
assert_fails "rejects malformed version override" normalize_version latest
assert_fails "rejects extra version components" normalize_version v1.2.3.4
assert_fails "rejects shell metacharacters in version" normalize_version 'v1.*.3'
assert_eq v2.3.4 \
  "$(version_from_release_url https://github.com/lexyurk/garmin-cli/releases/tag/v2.3.4)" \
  "reads version from the trusted latest release URL"
assert_fails "rejects an unexpected latest release host" \
  version_from_release_url https://example.com/lexyurk/garmin-cli/releases/tag/v2.3.4
assert_eq \
  https://github.com/lexyurk/garmin-cli/releases/download/v2.3.4/garmin-cli_2.3.4_linux_amd64.tar.gz \
  "$(release_download_url v2.3.4 garmin-cli_2.3.4_linux_amd64.tar.gz)" \
  "builds the release archive URL"

test_dir=$(mktemp -d "${TMPDIR:-/tmp}/garmin-cli-installer-test.XXXXXX")
trap 'rm -rf "$test_dir"' 0 HUP INT TERM

printf 'verified binary fixture\n' > "$test_dir/archive"
if command -v sha256sum >/dev/null 2>&1; then
  checksum=$(sha256sum "$test_dir/archive")
else
  checksum=$(shasum -a 256 "$test_dir/archive")
fi
checksum=${checksum%% *}
printf '%s  fixture.tar.gz\n' "$checksum" > "$test_dir/checksums.txt"
verify_checksum "$test_dir/archive" "$test_dir/checksums.txt" fixture.tar.gz
assert_eq verified verified "accepts a matching SHA256 checksum"

printf '%064d  fixture.tar.gz\n' 0 > "$test_dir/checksums.txt"
assert_fails "rejects a SHA256 checksum mismatch" \
  verify_checksum "$test_dir/archive" "$test_dir/checksums.txt" fixture.tar.gz

printf '#!/bin/sh\nprintf "fixture version\\n"\n' > "$test_dir/source-garmin"
install_binary "$test_dir/source-garmin" "$test_dir/custom/bin"
assert_eq "fixture version" "$($test_dir/custom/bin/garmin)" "installs an executable into a custom target"
assert_eq 0 "$(find "$test_dir/custom/bin" -name '.garmin.*' -type f | wc -l | tr -d ' ')" \
  "does not leave an install temporary file"

printf '1..%s\n' "$tests_run"
