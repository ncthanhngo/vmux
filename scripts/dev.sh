#!/usr/bin/env bash
# Local dev: build the universal sidecar, build the app (Debug), and launch it.
# The app embeds + spawns the sidecar itself; logs go to ~/Library/Logs/vmux/.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DERIVED_DATA="$REPO_ROOT/build/DerivedData"

"$REPO_ROOT/scripts/build-sidecar.sh"
CONFIGURATION=Debug DERIVED_DATA="$DERIVED_DATA" "$REPO_ROOT/scripts/build-app.sh"

APP_PATH="$DERIVED_DATA/Build/Products/Debug/Vmux.app"
echo "Launching $APP_PATH"
open "$APP_PATH"
echo "Sidecar log: tail -f ~/Library/Logs/vmux/sidecar.log"
