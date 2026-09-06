#!/usr/bin/env bash
# detect_env.sh — detect whether a project is Android, iOS, both, or neither,
# for JD AI Pay (外场AI付) integration routing.
#
# Usage:  bash detect_env.sh [project-root]   # defaults to current directory
#
# Output (stdout):
#   PLATFORM=<android|ios|both|none>
#   ANDROID_ROOT=<path or ->
#   IOS_ROOT=<path or ->
#   plus human-readable evidence lines prefixed with "  - ".
#
# Detection is marker-based and shallow (maxdepth) so it stays fast on big repos and
# handles cross-platform layouts (Flutter / React Native) that contain android/ and ios/.

set -uo pipefail

ROOT="${1:-.}"
if [ ! -d "$ROOT" ]; then
  echo "PLATFORM=none"
  echo "ANDROID_ROOT=-"
  echo "IOS_ROOT=-"
  echo "  - error: '$ROOT' is not a directory"
  exit 0
fi
ROOT="$(cd "$ROOT" && pwd)"

DEPTH=4   # how deep to look for project markers

android_evidence=()
ios_evidence=()
android_root="-"
ios_root="-"

# Directories to skip while scanning (build output / vendored deps).
PRUNE=( -name node_modules -o -name build -o -name '.git' -o -name Pods -o -name DerivedData -o -name '.gradle' )

# --- Android markers --------------------------------------------------------
# AndroidManifest.xml is the strongest signal; settings.gradle / gradlew also count.
while IFS= read -r f; do
  [ -z "$f" ] && continue
  android_evidence+=("$f")
done < <(find "$ROOT" -maxdepth "$DEPTH" \( "${PRUNE[@]}" \) -prune -o \
            \( -name AndroidManifest.xml -o -name settings.gradle -o -name settings.gradle.kts \
               -o -name gradlew \) -print 2>/dev/null | head -n 20)

if [ "${#android_evidence[@]}" -gt 0 ]; then
  # Prefer the directory containing a top-level settings.gradle / gradlew as the root;
  # else the parent of the shallowest AndroidManifest.xml.
  best=""
  for f in "${android_evidence[@]}"; do
    case "$f" in
      */settings.gradle|*/settings.gradle.kts|*/gradlew) best="$(dirname "$f")"; break ;;
    esac
  done
  if [ -z "$best" ]; then
    f="${android_evidence[0]}"
    # AndroidManifest is usually at <module>/src/main/AndroidManifest.xml → walk up to module root
    best="$(dirname "$f")"
    case "$best" in */src/main) best="$(dirname "$(dirname "$best")")" ;; esac
  fi
  android_root="$best"
fi

# --- iOS markers ------------------------------------------------------------
while IFS= read -r f; do
  [ -z "$f" ] && continue
  ios_evidence+=("$f")
done < <(find "$ROOT" -maxdepth "$DEPTH" \( "${PRUNE[@]}" \) -prune -o \
            \( -name '*.xcodeproj' -o -name '*.xcworkspace' -o -name 'Podfile' \
               -o -name 'AppDelegate.swift' -o -name 'AppDelegate.m' -o -name 'AppDelegate.mm' \) \
            -print 2>/dev/null | head -n 20)

if [ "${#ios_evidence[@]}" -gt 0 ]; then
  best=""
  for f in "${ios_evidence[@]}"; do
    case "$f" in
      *.xcworkspace) best="$(dirname "$f")"; break ;;
    esac
  done
  if [ -z "$best" ]; then
    for f in "${ios_evidence[@]}"; do
      case "$f" in *.xcodeproj) best="$(dirname "$f")"; break ;; esac
    done
  fi
  [ -z "$best" ] && best="$(dirname "${ios_evidence[0]}")"
  ios_root="$best"
fi

# --- Decide platform --------------------------------------------------------
have_android=0; [ "$android_root" != "-" ] && have_android=1
have_ios=0;     [ "$ios_root" != "-" ] && have_ios=1

if [ "$have_android" -eq 1 ] && [ "$have_ios" -eq 1 ]; then
  platform="both"
elif [ "$have_android" -eq 1 ]; then
  platform="android"
elif [ "$have_ios" -eq 1 ]; then
  platform="ios"
else
  platform="none"
fi

echo "PLATFORM=$platform"
echo "ANDROID_ROOT=$android_root"
echo "IOS_ROOT=$ios_root"

if [ "${#android_evidence[@]}" -gt 0 ]; then
  echo "  Android evidence:"
  for f in "${android_evidence[@]}"; do echo "  - $f"; done
fi
if [ "${#ios_evidence[@]}" -gt 0 ]; then
  echo "  iOS evidence:"
  for f in "${ios_evidence[@]}"; do echo "  - $f"; done
fi
if [ "$platform" = "none" ]; then
  echo "  - no Android or iOS project markers found under $ROOT (depth $DEPTH)"
  echo "  - ask the user which platform to integrate and where the project root is"
fi

# Also surface whether JD SDK binaries are already present (helps Step 2).
echo "  SDK binaries:"
aar="$(find "$ROOT" -maxdepth "$DEPTH" \( "${PRUNE[@]}" \) -prune -o -name '*.aar' -print 2>/dev/null | head -n 5)"
xcf="$(find "$ROOT" -maxdepth "$DEPTH" \( "${PRUNE[@]}" \) -prune -o -name 'JDPay.xcframework' -print 2>/dev/null | head -n 3)"
if [ -n "$aar" ]; then echo "$aar" | while IFS= read -r l; do echo "  - aar: $l"; done; else echo "  - aar: none found"; fi
if [ -n "$xcf" ]; then echo "$xcf" | while IFS= read -r l; do echo "  - xcframework: $l"; done; else echo "  - JDPay.xcframework: none found"; fi
