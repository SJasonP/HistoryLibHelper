# Changelog

All notable changes to HistoryLib Helper are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project follows
[Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-08-13

### Added

- A local-only desktop application for Windows, macOS, and Linux that exports browser history to HistoryLib `.hlz`
  archives.
- Automatic discovery of Google Chrome, Microsoft Edge, Brave, Vivaldi, Opera, Chromium, and Firefox history databases
  and their available browser histories.
- Read-only database access with private SQLite snapshots, preserving the original browser data and including
  uncheckpointed WAL records.
- Selection and combination of multiple browser histories into a single importable archive with prebuilt year, month,
  and day indexes.
- Optional password protection compatible with HistoryLib, using Argon2id password-based key derivation and
  authenticated XChaCha20-Poly1305 streaming encryption.
- Crash-safe output replacement so a failed or cancelled export does not overwrite an existing archive or leave a
  partial file at the selected destination.
- A Cobra-based `hlz-export` command-line interface for discovery, automatic export, and manually selected Chromium or
  Firefox databases.
- Complete American English and Simplified Chinese localization for the desktop interface.
- Light and dark themes with automatic, real-time system appearance tracking.
- Cross-platform build automation, application icons, third-party license notices, and macOS signing and notarization
  support.
