export type Locale = 'en-US' | 'zh-CN';
export type LanguageMode = 'auto' | Locale;

const enUS = {
    appName: 'HISTORYLIB HELPER',
    title: 'Export browser history',
    privacy: 'Reads local browser databases and creates one HistoryLib archive. Nothing is uploaded.',
    language: 'Language',
    automatic: 'Automatic',
    english: 'English',
    chinese: '简体中文',
    rescan: 'Rescan',
    profiles: 'Browser history',
    history: 'History',
    defaultHistory: 'Default history',
    manualHistory: 'Manually selected history',
    noProfiles: 'No history was found from Chrome, Edge, Brave, Vivaldi, Opera, Chromium, or Firefox.',
    output: 'Output',
    chooseOutput: 'Choose where to save the .hlz archive.',
    choose: 'Choose…',
    protection: 'Password protection',
    password: 'Password (optional)',
    confirmPassword: 'Confirm password',
    passwordHint: 'Leave both fields empty to create an archive without password protection.',
    weakPassword: 'For better security, use at least 12 characters.',
    passwordMismatch: 'The passwords do not match.',
    scanning: 'Scanning browser history…',
    found: (count: number) => `Found browser history in ${count} ${count === 1 ? 'location' : 'locations'}.`,
    snapshots: 'Creating consistent database snapshots…',
    exporting: 'Exporting…',
    exportProfiles: (count: number) => `Export history (${count})`,
    exported: (count: number, output: string) => `Exported ${count.toLocaleString('en-US')} records to ${output}`,
    error: (message: string) => `Error: ${message}`,
};

const zhCN: typeof enUS = {
    appName: 'HISTORYLIB HELPER',
    title: '导出浏览器历史记录',
    privacy: '读取本机浏览器数据库并创建一个 HistoryLib 归档。任何数据都不会上传。',
    language: '语言',
    automatic: '自动',
    english: 'English',
    chinese: '简体中文',
    rescan: '重新扫描',
    profiles: '浏览器历史记录',
    history: '历史记录',
    defaultHistory: '默认历史记录',
    manualHistory: '手动选择的历史记录',
    noProfiles: '未找到 Chrome、Edge、Brave、Vivaldi、Opera、Chromium 或 Firefox 的历史记录。',
    output: '输出位置',
    chooseOutput: '选择保存 .hlz 归档的位置。',
    choose: '选择…',
    protection: '密码保护',
    password: '密码（可选）',
    confirmPassword: '确认密码',
    passwordHint: '两个输入框均留空时，将创建不受密码保护的归档。',
    weakPassword: '为提高安全性，建议使用至少 12 个字符。',
    passwordMismatch: '两次输入的密码不一致。',
    scanning: '正在扫描浏览器历史记录…',
    found: (count: number) => `在 ${count} 个位置找到浏览器历史记录。`,
    snapshots: '正在创建一致的数据库快照…',
    exporting: '正在导出…',
    exportProfiles: (count: number) => `导出历史记录（${count}）`,
    exported: (count: number, output: string) => `已将 ${count.toLocaleString('zh-CN')} 条记录导出到 ${output}`,
    error: (message: string) => `错误：${message}`,
};

export type Messages = typeof enUS;

export function systemLocale(): Locale {
    return navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
}

export function resolveLocale(mode: LanguageMode): Locale {
    return mode === 'auto' ? systemLocale() : mode;
}

export function messages(locale: Locale): Messages {
    return locale === 'zh-CN' ? zhCN : enUS;
}
