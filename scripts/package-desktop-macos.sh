#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:-dev}"
APP_NAME="myclaw-desktop"
DMG_PATH="dist/${APP_NAME}-macos-${VERSION}.dmg"

rm -f "${DMG_PATH}"
mkdir -p dist

# Build with Wails
cd cmd/myclaw-desktop
wails build -platform darwin/universal -m -s
cd ../..

APP_SRC="cmd/myclaw-desktop/build/bin/${APP_NAME}.app"

# Sign if identity provided
if [[ -n "${CODESIGN_IDENTITY:-}" ]]; then
    codesign --force --deep --options runtime --sign "${CODESIGN_IDENTITY}" "${APP_SRC}"
fi

# Create DMG
DMG_TEMP=$(mktemp -d)
cp -R "${APP_SRC}" "${DMG_TEMP}/"
ln -s /Applications "${DMG_TEMP}/Applications"

hdiutil create \
    -volname "myclaw" \
    -srcfolder "${DMG_TEMP}" \
    -ov \
    -format UDZO \
    "${DMG_PATH}"

rm -rf "${DMG_TEMP}"
echo "Created ${DMG_PATH}"
