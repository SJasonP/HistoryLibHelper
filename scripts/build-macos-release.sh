#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${MACOS_RELEASE_ENV:-$ROOT_DIR/config/macos-release.env}"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

APP_NAME="${APP_NAME:-HistoryLibHelper}"
PLATFORM="${WAILS_PLATFORM:-darwin/universal}"
APP_PATH="${APP_PATH:-$ROOT_DIR/build/bin/$APP_NAME.app}"
ENTITLEMENTS="${ENTITLEMENTS:-$ROOT_DIR/build/darwin/entitlements.plist}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist/macos}"
ZIP_PATH="${ZIP_PATH:-$DIST_DIR/$APP_NAME-macOS.zip}"
SIGN_IDENTITY="${APPLE_SIGN_IDENTITY:-}"
NOTARY_PROFILE="${APPLE_NOTARY_KEYCHAIN_PROFILE:-}"
APPLE_ID_VALUE="${APPLE_ID:-}"
APPLE_TEAM_ID_VALUE="${APPLE_TEAM_ID:-}"
APPLE_APP_PASSWORD_VALUE="${APPLE_APP_PASSWORD:-}"
SKIP_BUILD="${SKIP_BUILD:-0}"
SKIP_NOTARIZE="${SKIP_NOTARIZE:-0}"

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

log() {
  printf '\n==> %s\n' "$*"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "$1 was not found on PATH"
}

discover_sign_identity() {
  security find-identity -v -p codesigning 2>/dev/null \
    | awk -F '"' '/Developer ID Application/ {print $2; exit}'
}

require_macos() {
  [[ "$(uname -s)" == "Darwin" ]] || die "macOS release signing must run on macOS"
}

require_notary_credentials() {
  if [[ -n "$NOTARY_PROFILE" ]]; then
    return
  fi
  if [[ -n "$APPLE_ID_VALUE" && -n "$APPLE_TEAM_ID_VALUE" && -n "$APPLE_APP_PASSWORD_VALUE" ]]; then
    return
  fi
  die "set APPLE_NOTARY_KEYCHAIN_PROFILE, or set APPLE_ID, APPLE_TEAM_ID, and APPLE_APP_PASSWORD"
}

build_app() {
  if [[ "$SKIP_BUILD" == "1" ]]; then
    log "Skipping Wails build"
    return
  fi
  log "Building $APP_NAME for $PLATFORM"
  CGO_ENABLED=1 wails build -clean -platform "$PLATFORM" -trimpath
}

sign_app() {
  [[ -d "$APP_PATH" ]] || die "app bundle not found: $APP_PATH"
  [[ -f "$ENTITLEMENTS" ]] || die "entitlements file not found: $ENTITLEMENTS"

  if [[ -z "$SIGN_IDENTITY" ]]; then
    SIGN_IDENTITY="$(discover_sign_identity || true)"
  fi
  [[ -n "$SIGN_IDENTITY" ]] || die "no Developer ID Application identity found; set APPLE_SIGN_IDENTITY"

  log "Signing $APP_PATH"
  xattr -cr "$APP_PATH"
  codesign \
    --force \
    --deep \
    --options runtime \
    --timestamp \
    --entitlements "$ENTITLEMENTS" \
    --sign "$SIGN_IDENTITY" \
    "$APP_PATH"

  log "Verifying code signature"
  codesign --verify --deep --strict --verbose=2 "$APP_PATH"
}

package_for_notary() {
  log "Creating notary archive"
  rm -rf "$DIST_DIR"
  mkdir -p "$DIST_DIR"
  ditto -c -k --keepParent "$APP_PATH" "$ZIP_PATH"
}

submit_notary() {
  if [[ "$SKIP_NOTARIZE" == "1" ]]; then
    log "Skipping notarization"
    return
  fi
  require_notary_credentials

  log "Submitting to Apple notary service"
  if [[ -n "$NOTARY_PROFILE" ]]; then
    xcrun notarytool submit "$ZIP_PATH" --keychain-profile "$NOTARY_PROFILE" --wait
  else
    xcrun notarytool submit "$ZIP_PATH" \
      --apple-id "$APPLE_ID_VALUE" \
      --team-id "$APPLE_TEAM_ID_VALUE" \
      --password "$APPLE_APP_PASSWORD_VALUE" \
      --wait
  fi

  log "Stapling notary ticket"
  xcrun stapler staple "$APP_PATH"
  xcrun stapler validate "$APP_PATH"

  log "Recreating archive with stapled app"
  rm -f "$ZIP_PATH"
  ditto -c -k --keepParent "$APP_PATH" "$ZIP_PATH"
}

assess_gatekeeper() {
  if [[ "$SKIP_NOTARIZE" == "1" ]]; then
    log "Skipping Gatekeeper notarization assessment"
    return
  fi
  log "Assessing with Gatekeeper"
  spctl -a -vvv -t install "$APP_PATH"
}

main() {
  require_macos
  require_command wails
  require_command security
  require_command codesign
  require_command xcrun
  require_command ditto
  require_command spctl

  build_app
  sign_app
  package_for_notary
  submit_notary
  assess_gatekeeper

  log "macOS release artifact ready"
  printf '%s\n' "$ZIP_PATH"
}

main "$@"
