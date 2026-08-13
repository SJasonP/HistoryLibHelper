# HistoryLib Helper

English | [简体中文](README.zh-CN.md)

HistoryLib Helper is a local-only Windows, macOS, and Linux utility that reads supported browser history databases and
exports HistoryLib `.hlz` v1 archives. It does not browse, edit, sync, or upload history.

## Supported sources

- Chromium databases: Google Chrome, Microsoft Edge, Brave, Vivaldi, Opera, and Chromium.
- Firefox `places.sqlite` databases.
- Automatically discovered browser history databases and manually selected databases through the CLI.

The source database is opened read-only and SQLite `VACUUM INTO` creates a consistent private snapshot. The original
browser database is never modified. If the browser prevents a safe snapshot, close it and retry.

## Technology

- Go 1.26
- Wails 2.14
- Cobra 1.10
- React 19.2
- TypeScript 7.0
- Vite 8.2
- modernc SQLite 1.56

Only stable releases are used.

## System requirements

- Windows 10 or later.
- macOS 12 Monterey or later.
- A current Linux distribution with GTK 3 and WebKitGTK 4.1.

## Development

```bash
go test ./...
wails dev
```

Build the GUI:

```bash
wails build -clean
```

On a current Linux distribution using WebKitGTK 4.1, add `-tags webkit2_41`.

Release helpers:

```bash
node scripts/generate-app-icons.mjs
node scripts/generate-third-party-notices.mjs
scripts/build-macos-release.sh
```

On Windows, run `scripts/build-windows-amd64.ps1` from PowerShell. Add `-Installer` to also create an NSIS installer.
The macOS release script reads signing and notarization values from environment variables or an ignored
`config/macos-release.env` file.

Build and use the CLI:

```bash
go build -o build/bin/hlz-export ./cmd/hlz-export
./build/bin/hlz-export list
./build/bin/hlz-export export --profile PROFILE_ID --output history.hlz
```

To create a password-protected archive without exposing the password in process arguments:

```bash
printf '%s\n' 'your password' | ./build/bin/hlz-export export \
  --profile PROFILE_ID \
  --output history.hlz \
  --password-stdin
```

For a manually selected database:

```bash
./build/bin/hlz-export export \
  --database /path/to/History \
  --engine chromium \
  --browser "Google Chrome" \
  --output history.hlz
```

## Third-party notices

Third-party open-source components remain subject to their respective license terms. The complete component list and
license texts are available in [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

Browser and platform names are trademarks of their respective owners. HistoryLib Helper is not affiliated with or
endorsed by those owners.