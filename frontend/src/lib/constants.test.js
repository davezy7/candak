import test from 'node:test';
import assert from 'node:assert/strict';
import {
  DOWNLOAD_STATUS,
  CHUNK_STATUS,
  FILTER_MODE,
  SORT_MODE,
  IPC_EVENTS,
  STORAGE_KEYS
} from './constants.js';

test('constants are frozen and immutable', () => {
  assert.ok(Object.isFrozen(DOWNLOAD_STATUS));
  assert.ok(Object.isFrozen(CHUNK_STATUS));
  assert.ok(Object.isFrozen(FILTER_MODE));
  assert.ok(Object.isFrozen(SORT_MODE));
  assert.ok(Object.isFrozen(IPC_EVENTS));
  assert.ok(Object.isFrozen(STORAGE_KEYS));

  assert.throws(() => {
    DOWNLOAD_STATUS.NEW_STATUS = 'hacked';
  }, TypeError);
});

test('DOWNLOAD_STATUS matches backend lifecycle models', () => {
  assert.equal(DOWNLOAD_STATUS.QUEUED, 'queued');
  assert.equal(DOWNLOAD_STATUS.DOWNLOADING, 'downloading');
  assert.equal(DOWNLOAD_STATUS.PAUSED, 'paused');
  assert.equal(DOWNLOAD_STATUS.COMPLETED, 'completed');
  assert.equal(DOWNLOAD_STATUS.ERROR, 'error');
  assert.equal(DOWNLOAD_STATUS.CANCELLED, 'cancelled');
});
