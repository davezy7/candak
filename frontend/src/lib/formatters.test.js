import test from 'node:test';
import assert from 'node:assert/strict';
import { getExtensionBadge, getFolderPath, formatBytes } from './formatters.js';

test('getExtensionBadge returns uppercase extension', () => {
  assert.equal(getExtensionBadge('test.zip'), 'ZIP');
  assert.equal(getExtensionBadge('archive.tar.gz'), 'GZ');
  assert.equal(getExtensionBadge('image.JPEG'), 'JPEG');
  assert.equal(getExtensionBadge('video.mp4'), 'MP4');
});

test('getExtensionBadge truncates extensions longer than 5 chars to 4 chars', () => {
  assert.equal(getExtensionBadge('file.torrent'), 'TORR');
  assert.equal(getExtensionBadge('data.package'), 'PACK');
});

test('getExtensionBadge handles edge cases and fallbacks', () => {
  assert.equal(getExtensionBadge(''), 'FILE');
  assert.equal(getExtensionBadge(null), 'FILE');
  assert.equal(getExtensionBadge('noextension'), 'FILE');
  assert.equal(getExtensionBadge('', '/home/user/test.iso'), 'ISO');
  assert.equal(getExtensionBadge('noext', '/home/user/test.dmg'), 'DMG');
  assert.equal(getExtensionBadge('noext', '/home/john.doe/Downloads/noext'), 'FILE');
  assert.equal(getExtensionBadge('', 'C:\\Users\\john.doe\\Downloads\\noext'), 'FILE');
});

test('getFolderPath extracts directory path', () => {
  assert.equal(getFolderPath('/Users/diksha/Downloads/file.zip'), '/Users/diksha/Downloads');
  assert.equal(getFolderPath('/file.zip'), '/');
  assert.equal(getFolderPath('C:\\Users\\user\\Downloads\\app.exe'), 'C:\\Users\\user\\Downloads');
  assert.equal(getFolderPath('C:\\file.zip'), 'C:\\');
  assert.equal(getFolderPath(''), '');
  assert.equal(getFolderPath('file.zip'), '');
});

test('formatBytes formats sizes accurately', () => {
  assert.equal(formatBytes(0), '0 B');
  assert.equal(formatBytes(1024), '1 KB');
  assert.equal(formatBytes(1024 * 1024 * 1.5), '1.5 MB');
});
