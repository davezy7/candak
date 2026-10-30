/**
 * Canonical domain constants and lifecycle enums.
 * Mirrors Go backend definitions in internal/models/models.go.
 */

export const DOWNLOAD_STATUS = Object.freeze({
  QUEUED: 'queued',
  DOWNLOADING: 'downloading',
  PAUSED: 'paused',
  COMPLETED: 'completed',
  ERROR: 'error',
  CANCELLED: 'cancelled'
});

export const CHUNK_STATUS = Object.freeze({
  PENDING: 'pending',
  DOWNLOADING: 'downloading',
  COMPLETED: 'completed',
  ERROR: 'error'
});

export const FILTER_MODE = Object.freeze({
  ALL: 'all',
  DOWNLOADING: 'downloading',
  COMPLETED: 'completed',
  PAUSED: 'paused'
});

export const SORT_MODE = Object.freeze({
  NEWEST: 'newest',
  OLDEST: 'oldest',
  NAME: 'name',
  SIZE: 'size'
});

export const IPC_EVENTS = Object.freeze({
  PROGRESS: 'download:progress',
  STATUS_CHANGE: 'download:status_change',
  DELETED: 'download:deleted',
  GRABBED: 'download:grabbed',
  PROMPT_MODE: 'download:prompt_mode'
});

export const STORAGE_KEYS = Object.freeze({
  THEME: 'candak_theme',
  CLIPBOARD_WATCHING: 'candak_clipboard_watching',
  LAST_UPDATE_CHECK: 'candak_last_update_check',
  LOCALE: 'candak_locale',
  CUSTOM_DOWNLOAD_DIR: 'candak_custom_download_dir',
  DEFAULT_CONNECTIONS: 'candak_default_connections',
  ACCEPTED_EXTENSIONS: 'candak_accepted_extensions',
  ENFORCE_EXTENSIONS: 'candak_enforce_extensions'
});
