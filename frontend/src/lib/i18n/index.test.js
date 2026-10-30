import test from 'node:test';
import assert from 'node:assert/strict';
import { translate } from './index.js';

test('translate resolves simple dot-notation keys', () => {
  const result = translate('en-US', 'common.save');
  assert.equal(result, 'Save');
});

test('translate handles single-brace token interpolation', () => {
  const result = translate('en-US', 'table.footer.active_done_summary', {
    active: 3,
    done: 12
  });
  assert.equal(result, '3 active · 12 done');
});

test('translate handles pluralization dispatch (_one vs _other)', () => {
  const singular = translate('en-US', 'table.selection.selected_count', { count: 1 });
  assert.equal(singular, '1 download selected');

  const plural = translate('en-US', 'table.selection.selected_count', { count: 5 });
  assert.equal(plural, '5 downloads selected');
});

test('translate falls back to raw key if missing in all locales', () => {
  const missing = translate('en-US', 'nonexistent.section.key');
  assert.equal(missing, 'nonexistent.section.key');
});

test('translate resolves Indonesian locale and falls back to English when key missing', () => {
  assert.equal(translate('id', 'common.save'), 'Simpan');
  assert.equal(translate('id', 'modals.history.title'), 'Riwayat Unduhan');
  assert.equal(translate('id', 'table.navigation.add_download'), 'TAMBAH UNDUHAN');
});
