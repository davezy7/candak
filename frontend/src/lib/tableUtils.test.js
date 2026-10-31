import test from 'node:test';
import assert from 'node:assert/strict';
import { sortDownloads, filterDownloads, updateSelection } from './tableUtils.js';

test('sortDownloads sorts by newest using ISO createdAt and string IDs', () => {
  const items = [
    { id: 'uuid-1', filename: 'b.zip', createdAt: '2026-10-30T10:00:00Z', totalSize: 100 },
    { id: 'uuid-2', filename: 'a.zip', createdAt: '2026-10-30T12:00:00Z', totalSize: 200 },
    { id: 'uuid-3', filename: 'c.zip', createdAt: '2026-10-30T09:00:00Z', totalSize: 50 },
  ];

  const newest = sortDownloads(items, 'newest');
  assert.deepEqual(newest.map(d => d.id), ['uuid-2', 'uuid-1', 'uuid-3']);

  const oldest = sortDownloads(items, 'oldest');
  assert.deepEqual(oldest.map(d => d.id), ['uuid-3', 'uuid-1', 'uuid-2']);

  const bySize = sortDownloads(items, 'size');
  assert.deepEqual(bySize.map(d => d.id), ['uuid-2', 'uuid-1', 'uuid-3']);

  const byName = sortDownloads(items, 'name');
  assert.deepEqual(byName.map(d => d.id), ['uuid-2', 'uuid-1', 'uuid-3']);
});

test('filterDownloads filters by status and search substring', () => {
  const items = [
    { id: '1', filename: 'ubuntu-24.04.iso', url: 'https://releases.ubuntu.com', status: 'downloading' },
    { id: '2', filename: 'fedora-40.iso', url: 'https://fedoraproject.org', status: 'completed' },
    { id: '3', filename: 'archlinux.iso', url: 'https://archlinux.org', status: 'paused' },
    { id: '4', filename: 'debian-12.iso', url: 'https://debian.org', status: 'queued' }
  ];

  assert.equal(filterDownloads(items, 'completed').length, 1);
  assert.equal(filterDownloads(items, 'downloading').length, 2); // includes downloading and queued
  assert.equal(filterDownloads(items, 'all', 'fedora').length, 1);
  assert.equal(filterDownloads(items, 'all', 'UBUNTU').length, 1);
  assert.equal(filterDownloads(items, 'completed', 'arch').length, 0);
  assert.equal(filterDownloads(items, 'paused').length, 1);
});

test('updateSelection handles single click exclusive selection', () => {
  const res = updateSelection({
    selectedIds: new Set(['item-1']),
    lastSelectedId: 'item-1',
    targetId: 'item-2'
  });
  assert.deepEqual(Array.from(res.nextSelectedIds), ['item-2']);
  assert.equal(res.nextLastId, 'item-2');
});

test('updateSelection handles checkbox / modifier key toggle', () => {
  const s1 = updateSelection({
    selectedIds: new Set(['item-1']),
    lastSelectedId: 'item-1',
    targetId: 'item-2',
    isCheckbox: true
  });
  assert.deepEqual(Array.from(s1.nextSelectedIds), ['item-1', 'item-2']);
  assert.equal(s1.nextLastId, 'item-2');

  const s2 = updateSelection({
    selectedIds: s1.nextSelectedIds,
    lastSelectedId: s1.nextLastId,
    targetId: 'item-1',
    modifierKey: true
  });
  assert.deepEqual(Array.from(s2.nextSelectedIds), ['item-2']);
});

test('updateSelection handles shift-click range selection', () => {
  const visible = [
    { id: 'a' },
    { id: 'b' },
    { id: 'c' },
    { id: 'd' }
  ];

  const res = updateSelection({
    selectedIds: new Set(['a']),
    lastSelectedId: 'a',
    targetId: 'c',
    shiftKey: true,
    visibleList: visible
  });

  assert.deepEqual(Array.from(res.nextSelectedIds), ['a', 'b', 'c']);
  assert.equal(res.nextLastId, 'c');

  // Shift + Cmd/Ctrl preserves existing selection outside the range
  const additiveRes = updateSelection({
    selectedIds: new Set(['x', 'a']),
    lastSelectedId: 'a',
    targetId: 'c',
    shiftKey: true,
    modifierKey: true,
    visibleList: visible
  });
  assert.deepEqual(Array.from(additiveRes.nextSelectedIds), ['x', 'a', 'b', 'c']);
  assert.equal(additiveRes.nextLastId, 'c');
});

test('updateSelection sticky mode toggles on click when multiple items selected', () => {
  const res = updateSelection({
    selectedIds: new Set(['a', 'b']),
    lastSelectedId: 'b',
    targetId: 'c'
  });
  assert.deepEqual(Array.from(res.nextSelectedIds), ['a', 'b', 'c']);

  const deselectRes = updateSelection({
    selectedIds: new Set(['a', 'b', 'c']),
    lastSelectedId: 'c',
    targetId: 'b'
  });
  assert.deepEqual(Array.from(deselectRes.nextSelectedIds), ['a', 'c']);
});
