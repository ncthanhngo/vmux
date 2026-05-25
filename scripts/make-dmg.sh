#!/usr/bin/env bash
# Produce a signed + notarized DMG.
# Requires Developer ID env vars:
#   VMUX_SIGN_IDENTITY  - "Developer ID Application: Name (TEAMID)"
#   VMUX_TEAM_ID        - Apple Developer team id
#   VMUX_NOTARY_PROFILE - notarytool keychain profile name
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$REPO_ROOT/app"
DIST_DIR="$REPO_ROOT/dist"
DERIVED_DATA="$REPO_ROOT/build/DerivedData"
VERSION="${VMUX_VERSION:-0.0.0}"

: "${VMUX_SIGN_IDENTITY:?set VMUX_SIGN_IDENTITY (Developer ID Application: …)}"
: "${VMUX_NOTARY_PROFILE:?set VMUX_NOTARY_PROFILE (xcrun notarytool store-credentials profile)}"

command -v create-dmg >/dev/null || { echo "create-dmg missing: brew install create-dmg"; exit 1; }

# 1. Build Release with the universal sidecar embedded.
VMUX_VERSION="$VERSION" "$REPO_ROOT/scripts/build-sidecar.sh"
CONFIGURATION=Release "$REPO_ROOT/scripts/build-app.sh"

APP_PATH="$DERIVED_DATA/Build/Products/Release/Vmux.app"
SIDECAR="$APP_PATH/Contents/Resources/vmux-sidecar"

# 2. Sign inside-out: nested binary first, then the bundle. No --deep (Apple
#    discourages it for notarization; sign each component explicitly).
echo "Signing sidecar + app…"
codesign --force --options runtime --timestamp \
  --sign "$VMUX_SIGN_IDENTITY" "$SIDECAR"
codesign --force --options runtime --timestamp \
  --sign "$VMUX_SIGN_IDENTITY" "$APP_PATH"
codesign --verify --strict --verbose=2 "$APP_PATH"

# 3. Build the DMG.
mkdir -p "$DIST_DIR"
DMG_PATH="$DIST_DIR/vmux-$VERSION.dmg"
rm -f "$DMG_PATH"
create-dmg \
  --volname "vmux $VERSION" \
  --app-drop-link 480 180 \
  --icon "Vmux.app" 160 180 \
  --window-size 660 400 \
  "$DMG_PATH" "$APP_PATH"

# 4. Notarize + staple.
echo "Notarizing…"
xcrun notarytool submit "$DMG_PATH" --keychain-profile "$VMUX_NOTARY_PROFILE" --wait
xcrun stapler staple "$DMG_PATH"

echo "Done: $DMG_PATH"
