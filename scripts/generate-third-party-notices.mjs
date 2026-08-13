import {execSync} from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

function readJSON(filePath) {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function uniqueBy(items, keyFn) {
    const seen = new Set();
    const result = [];
    for (const item of items) {
        const key = keyFn(item);
        if (seen.has(key)) {
            continue;
        }
        seen.add(key);
        result.push(item);
    }
    return result;
}

function detectLicenseText(text) {
    const normalized = text.toLowerCase();
    const found = [];
    if (normalized.includes('mozilla public license version 2.0')) found.push('MPL-2.0');
    if (normalized.includes('apache license') && normalized.includes('version 2.0')) found.push('Apache-2.0');
    if (normalized.includes('mit license') || (normalized.includes('permission is hereby granted, free of charge') && normalized.includes('software'))) found.push('MIT');
    if (normalized.includes('redistribution and use in source and binary forms')) found.push('BSD-style');
    if (normalized.includes('isc license') || normalized.includes('permission to use, copy, modify, and/or distribute this software')) found.push('ISC');
    if (normalized.includes('the unlicense')) found.push('Unlicense');
    return [...new Set(found)].join(' / ') || 'unknown';
}

function licenseFiles(packageDir) {
    const names = fs.readdirSync(packageDir).filter((name) => /^(license|licence|copying|copyright|notice)(\.|$|[-_])/i.test(name));
    return names.sort().map((name) => {
        const filePath = path.join(packageDir, name);
        return {name, text: fs.readFileSync(filePath, 'utf8').trim()};
    });
}

function goRuntimeComponents() {
    const template = '{{with .Module}}{{printf "%s\\t%s\\t%s" .Path .Version .Dir}}{{end}}';
    const output = execSync(`go list -deps -test=false -f '${template}' . ./cmd/hlz-export`, {
        cwd: repoRoot,
        encoding: 'utf8',
    });
    return uniqueBy(output
        .split('\n')
        .filter(Boolean)
        .map((line) => {
            const [name, version, dir] = line.split('\t');
            return {ecosystem: 'Go', name, version, dir};
        })
        .filter((component) => component.name !== 'historylibhelper'), (component) => component.name)
        .sort((left, right) => left.name.localeCompare(right.name));
}

function frontendPackageDependencies() {
    const packageJSON = readJSON(path.join(repoRoot, 'frontend/package.json'));
    const lock = readJSON(path.join(repoRoot, 'frontend/package-lock.json'));
    return Object.keys(packageJSON.dependencies ?? {})
        .sort()
        .map((name) => {
            const meta = lock.packages?.[`node_modules/${name}`] ?? {};
            return {
                ecosystem: 'npm',
                name,
                version: meta.version ?? '',
                license: meta.license ?? 'unknown',
                dir: path.join(repoRoot, 'frontend/node_modules', name),
            };
        });
}

function frontendNotableBuildDependencies() {
    const lock = readJSON(path.join(repoRoot, 'frontend/package-lock.json'));
    return Object.entries(lock.packages ?? {})
        .filter(([packagePath]) => packagePath.startsWith('node_modules/'))
        .map(([packagePath, meta]) => ({
            name: packagePath.slice('node_modules/'.length),
            version: meta.version ?? '',
            license: meta.license ?? 'unknown',
        }))
        .filter((component) => component.license !== 'MIT')
        .sort((left, right) => left.name.localeCompare(right.name));
}

function componentLicense(component) {
    if (component.license) {
        return component.license;
    }
    try {
        const files = licenseFiles(component.dir);
        return detectLicenseText(files.map((file) => file.text).join('\n'));
    } catch {
        return 'unknown';
    }
}

function markdownTable(components, includeNotes = false) {
    const header = includeNotes
        ? ['| Component | Version | License | Notes |', '|---|---:|---|---|']
        : ['| Component | Version | License |', '|---|---:|---|'];
    const rows = components.map((component) => {
        const name = `\`${component.name}\``;
        const version = component.version ? `\`${component.version}\`` : '';
        const license = componentLicense(component);
        if (includeNotes) {
            return `| ${name} | ${version} | ${license} | ${component.notes ?? ''} |`;
        }
        return `| ${name} | ${version} | ${license} |`;
    });
    return [...header, ...rows].join('\n');
}

function noticeBlocks(components) {
    const blocks = [];
    for (const component of components) {
        const files = licenseFiles(component.dir);
        for (const file of files) {
            blocks.push([`### ${component.name} ${component.version} - ${file.name}`, '', '```text', file.text, '```'].join('\n'));
        }
    }
    return blocks.join('\n\n');
}

const goComponents = goRuntimeComponents();
const frontendComponents = frontendPackageDependencies();
const notableBuildDependencies = [...frontendNotableBuildDependencies().map((component) => ({
    ...component,
    notes: 'Frontend source/build dependency from package-lock.json.',
})), {
    name: 'github.com/hashicorp/golang-lru/v2',
    version: 'v2.0.7',
    license: 'MPL-2.0',
    notes: 'Appears in the Go module graph through modernc.org tooling/test paths; not in the current runtime dependency graph.',
}].sort((left, right) => left.name.localeCompare(right.name));

const markdown = `# Third-Party Notices

This document summarizes third-party open source components used by HistoryLibHelper.

HistoryLibHelper's own source code is licensed under the MIT License in \`LICENSE\`. Third-party components are licensed under their own license terms. This notice is provided for release compliance and attribution; it is not legal advice.

## Runtime Components

These components are part of the current Go application dependency graph for the CLI and Wails desktop application, based on \`go list -deps -test=false\`.

${markdownTable(goComponents)}

The frontend is bundled into the desktop application. Its package dependency manifest currently includes:

${markdownTable(frontendComponents)}

## Source And Build Dependencies

The source repository also contains dependency manifests for development and build tools:

- Go dependencies: \`go.mod\` and \`go.sum\`.
- Frontend dependencies: \`frontend/package.json\` and \`frontend/package-lock.json\`.

Notable non-MIT licenses in the build/source dependency set include:

${markdownTable(notableBuildDependencies, true)}

If a release package includes source dependencies, \`node_modules\`, a Go module cache, or build tool binaries, include the relevant upstream license files for those packaged components as well.

## Runtime License Notices

The following license files were copied from the runtime package directories used to build this release notice.

${noticeBlocks([...goComponents, ...frontendComponents])}
`;

fs.writeFileSync(path.join(repoRoot, 'THIRD-PARTY-NOTICES.md'), markdown);
