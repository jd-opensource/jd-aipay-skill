#!/usr/bin/env bash
# download_sdk.sh — Download and extract JD AI Pay SDK zip to the project root.
#
# Usage:  bash download_sdk.sh <url_or_path> [target_dir]
#   url_or_path : HTTP(S) URL or local zip file path (required)
#   target_dir  : directory to extract into (defaults to current directory)
#
# Output (stdout):
#   EXTRACT_DIR=<path>
#   SDK files found after extraction (*.aar / *.xcframework)

set -uo pipefail

SOURCE="${1:-}"
TARGET="${2:-.}"

if [ -z "$SOURCE" ]; then
  echo "ERROR: no download URL or local zip path provided"
  echo "Usage: bash download_sdk.sh <url_or_path> [target_dir]"
  exit 1
fi

if [ ! -d "$TARGET" ]; then
  echo "ERROR: target directory '$TARGET' does not exist"
  exit 1
fi

TARGET="$(cd "$TARGET" && pwd)"
TMP_ZIP="$TARGET/.jdaipay_sdk_download_$$.zip"
CLEANUP_ZIP=0

# --- Download or locate the zip -----------------------------------------------

if [[ "$SOURCE" =~ ^https?:// ]]; then
  echo "Downloading SDK from: $SOURCE"
  if ! curl -fSL --progress-bar -o "$TMP_ZIP" "$SOURCE"; then
    echo "ERROR: download failed from $SOURCE"
    rm -f "$TMP_ZIP"
    exit 1
  fi
  CLEANUP_ZIP=1
  echo "Download complete: $(du -h "$TMP_ZIP" | cut -f1)"
else
  if [ ! -f "$SOURCE" ]; then
    echo "ERROR: local file not found: $SOURCE"
    exit 1
  fi
  TMP_ZIP="$(cd "$(dirname "$SOURCE")" && pwd)/$(basename "$SOURCE")"
fi

# --- Extract -------------------------------------------------------------------

echo "Extracting to: $TARGET"
if ! unzip -o -q "$TMP_ZIP" -d "$TARGET"; then
  echo "ERROR: unzip failed"
  [ "$CLEANUP_ZIP" -eq 1 ] && rm -f "$TMP_ZIP"
  exit 1
fi

# Clean up downloaded zip (keep local zips untouched)
[ "$CLEANUP_ZIP" -eq 1 ] && rm -f "$TMP_ZIP"

echo "EXTRACT_DIR=$TARGET"

# --- Report SDK files found ----------------------------------------------------

echo "  SDK files found after extraction:"
aar="$(find "$TARGET" -maxdepth 4 -name '*.aar' 2>/dev/null)"
xcf="$(find "$TARGET" -maxdepth 4 -name 'JDPay.xcframework' -type d 2>/dev/null)"

if [ -n "$aar" ]; then
  echo "$aar" | while IFS= read -r l; do echo "  - aar: $l"; done
else
  echo "  - aar: none found"
fi

if [ -n "$xcf" ]; then
  echo "$xcf" | while IFS= read -r l; do echo "  - xcframework: $l"; done
else
  echo "  - xcframework: none found"
fi

if [ -z "$aar" ] && [ -z "$xcf" ]; then
  echo "  WARNING: no SDK binaries (*.aar / JDPay.xcframework) found after extraction"
  echo "  Please verify the zip contents are correct"
  exit 0
fi

echo "Done."
