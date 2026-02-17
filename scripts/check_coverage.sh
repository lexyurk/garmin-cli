#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   scripts/check_coverage.sh <min_percent> [coverage_profile]
#
# Example:
#   scripts/check_coverage.sh 80 coverage.out

min="${1:-80}"
profile="${2:-coverage.out}"

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 2
fi

total="$(
  go tool cover -func="$profile" \
    | awk '/^total:/ { gsub(/%/, "", $3); print $3 }'
)"

if [[ -z "$total" ]]; then
  echo "failed to parse total coverage from $profile" >&2
  exit 2
fi

awk -v total="$total" -v min="$min" 'BEGIN {
  if ((total + 0) < (min + 0)) {
    printf("total coverage %.1f%% is below minimum %.1f%%\n", total, min) > "/dev/stderr"
    exit 1
  }
  printf("total coverage %.1f%% meets minimum %.1f%%\n", total, min)
}'

