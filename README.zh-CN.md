# HistoryLib Helper

[English](README.md) | 简体中文

HistoryLib Helper 是一个仅在本机运行的 Windows、macOS 和 Linux 工具。它读取受支持浏览器的历史记录数据库并导出 HistoryLib
`.hlz` v1 归档，不提供浏览、编辑或同步功能，也不会上传历史记录。

## 支持的数据源

- Chromium 数据库：Google Chrome、Microsoft Edge、Brave、Vivaldi、Opera 和 Chromium。
- Firefox `places.sqlite` 数据库。
- GUI 自动发现浏览器历史记录数据库；CLI 还可以手动指定数据库。

程序以只读方式打开源数据库，并使用 SQLite `VACUUM INTO` 创建一致的私有快照，绝不会修改浏览器的原始数据库。如果浏览器阻止安全快照，请退出浏览器后重试。

## 技术栈

- Go 1.26
- Wails 2.14
- Cobra 1.10
- React 19.2
- TypeScript 7.0
- Vite 8.2
- modernc SQLite 1.56

仅使用正式稳定版本。

## 系统要求

- Windows 10 或更高版本。
- macOS 12 Monterey 或更高版本。
- 配备 GTK 3 和 WebKitGTK 4.1 的较新 Linux 发行版。

## 开发

```bash
go test ./...
wails dev
```

构建 GUI：

```bash
wails build -clean
```

在使用 WebKitGTK 4.1 的现代 Linux 发行版上，请添加 `-tags webkit2_41`。

发布辅助脚本：

```bash
node scripts/generate-app-icons.mjs
node scripts/generate-third-party-notices.mjs
scripts/build-macos-release.sh
```

Windows 上请在 PowerShell 中运行 `scripts/build-windows-amd64.ps1`；添加 `-Installer` 可同时创建 NSIS 安装程序。macOS
发布脚本会从环境变量或被 Git 忽略的
`config/macos-release.env` 文件读取签名和公证配置。

构建和使用 CLI：

```bash
go build -o build/bin/hlz-export ./cmd/hlz-export
./build/bin/hlz-export list
./build/bin/hlz-export export --profile PROFILE_ID --output history.hlz
```

安全地从标准输入读取密码并创建受密码保护的归档：

```bash
printf '%s\n' '你的密码' | ./build/bin/hlz-export export \
  --profile PROFILE_ID \
  --output history.hlz \
  --password-stdin
```

手动指定数据库：

```bash
./build/bin/hlz-export export \
  --database /path/to/History \
  --engine chromium \
  --browser "Google Chrome" \
  --output history.hlz
```

## 第三方声明

项目使用的第三方开源组件分别适用其各自的许可证条款。完整组件列表及许可证文本请查看
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md)。

文中提及的浏览器及平台名称是其各自所有者的商标。HistoryLib Helper 与这些所有者不存在隶属或认可关系。