#!/usr/bin/env bash
# Build the Vmux macOS app. Regenerates the Xcode project from project.yml,
# then runs xcodebuild. CONFIGURATION defaults to Debug; pass Release for dist.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$REPO_ROOT/app"
CONFIGURATION="${CONFIGURATION:-Debug}"
DERIVED_DATA="${DERIVED_DATA:-$REPO_ROOT/build/DerivedData}"

# Sidecar must exist before the app's embed phase runs.
if [ ! -f "$REPO_ROOT/sidecar/bin/vmux-sidecar" ]; then
  echo "Sidecar binary missing — building it first…"
  "$REPO_ROOT/scripts/build-sidecar.sh"
fi

echo "Generating Xcode project…"
( cd "$APP_DIR" && xcodegen generate )

echo "Building Vmux ($CONFIGURATION)…"
xcodebuild \
  -project "$APP_DIR/Vmux.xcodeproj" \
  -scheme Vmux \
  -configuration "$CONFIGURATION" \
  -derivedDataPath "$DERIVED_DATA" \
  CODE_SIGN_IDENTITY="-" CODE_SIGNING_REQUIRED=NO CODE_SIGNING_ALLOWED=NO \
  build

APP_PATH="$DERIVED_DATA/Build/Products/$CONFIGURATION/Vmux.app"
echo "Built: $APP_PATH"
