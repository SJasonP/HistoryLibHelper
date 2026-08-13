import {execFileSync} from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const buildDir = path.join(repoRoot, 'build');
const sourcePNG = path.join(buildDir, 'appicon.png');
const iconsetDir = path.join(buildDir, 'darwin', 'icon.iconset');
const macIcon = path.join(buildDir, 'darwin', 'icon.icns');
const winDir = path.join(buildDir, 'windows');
const winIcon = path.join(winDir, 'icon.ico');
const linuxDir = path.join(buildDir, 'linux');
const linuxIcon = path.join(linuxDir, 'icon.png');

const macSizes = [['icon_16x16.png', 16], ['icon_16x16@2x.png', 32], ['icon_32x32.png', 32], ['icon_32x32@2x.png', 64], ['icon_128x128.png', 128], ['icon_128x128@2x.png', 256], ['icon_256x256.png', 256], ['icon_256x256@2x.png', 512], ['icon_512x512.png', 512], ['icon_512x512@2x.png', 1024]];
const icoSizes = [16, 24, 32, 48, 64, 128, 256];

function run(command, args) {
    execFileSync(command, args, {stdio: 'inherit'});
}

function ensureDir(dir) {
    fs.mkdirSync(dir, {recursive: true});
}

function resizePNG(size, outputPath) {
    run('sips', ['-z', String(size), String(size), sourcePNG, '--out', outputPath]);
}

function createMacIcon() {
    fs.rmSync(iconsetDir, {recursive: true, force: true});
    ensureDir(iconsetDir);
    for (const [name, size] of macSizes) {
        resizePNG(size, path.join(iconsetDir, name));
    }
    fs.rmSync(macIcon, {force: true});
    run('iconutil', ['-c', 'icns', iconsetDir, '-o', macIcon]);
    fs.rmSync(iconsetDir, {recursive: true, force: true});
}

function makeICO(entries) {
    const headerSize = 6;
    const directorySize = 16 * entries.length;
    let offset = headerSize + directorySize;
    const chunks = [Buffer.alloc(headerSize)];
    chunks[0].writeUInt16LE(0, 0);
    chunks[0].writeUInt16LE(1, 2);
    chunks[0].writeUInt16LE(entries.length, 4);

    for (const entry of entries) {
        const dir = Buffer.alloc(16);
        dir.writeUInt8(entry.size >= 256 ? 0 : entry.size, 0);
        dir.writeUInt8(entry.size >= 256 ? 0 : entry.size, 1);
        dir.writeUInt8(0, 2);
        dir.writeUInt8(0, 3);
        dir.writeUInt16LE(1, 4);
        dir.writeUInt16LE(32, 6);
        dir.writeUInt32LE(entry.data.length, 8);
        dir.writeUInt32LE(offset, 12);
        chunks.push(dir);
        offset += entry.data.length;
    }
    chunks.push(...entries.map((entry) => entry.data));
    return Buffer.concat(chunks);
}

function createWindowsIcon() {
    ensureDir(winDir);
    const entries = icoSizes.map((size) => {
        const pngPath = path.join(winDir, `icon-${size}.png`);
        resizePNG(size, pngPath);
        return {size, data: fs.readFileSync(pngPath)};
    });
    fs.writeFileSync(winIcon, makeICO(entries));
    for (const size of icoSizes) {
        fs.rmSync(path.join(winDir, `icon-${size}.png`), {force: true});
    }
}

function createLinuxIcon() {
    ensureDir(linuxDir);
    fs.copyFileSync(sourcePNG, linuxIcon);
}

if (!fs.existsSync(sourcePNG)) {
    throw new Error(`Missing source PNG: ${sourcePNG}`);
}

createMacIcon();
createWindowsIcon();
createLinuxIcon();

console.log('Generated app icons:');
console.log(path.relative(repoRoot, sourcePNG));
console.log(path.relative(repoRoot, macIcon));
console.log(path.relative(repoRoot, winIcon));
console.log(path.relative(repoRoot, linuxIcon));
