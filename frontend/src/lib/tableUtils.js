/**
 * Pure utility functions for TableView: sorting, filtering, selection.
 */
import { DOWNLOAD_STATUS, FILTER_MODE, SORT_MODE } from './constants.js';

export function sortDownloads(list, sortBy = SORT_MODE.NEWEST) {
  return [...list].sort((a, b) => {
    if (sortBy === SORT_MODE.NAME) {
      return (a.filename || '').localeCompare(b.filename || '');
    }
    if (sortBy === SORT_MODE.SIZE) {
      return (b.totalSize || 0) - (a.totalSize || 0);
    }
    const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    if (sortBy === SORT_MODE.OLDEST) {
      if (timeA !== timeB) return timeA - timeB;
      return String(a.id || '').localeCompare(String(b.id || ''));
    }
    // 'newest' default
    if (timeA !== timeB) return timeB - timeA;
    return String(b.id || '').localeCompare(String(a.id || ''));
  });
}

export function filterDownloads(list, filter = FILTER_MODE.ALL, searchQuery = '') {
  return list
    .filter(d => {
      if (filter === FILTER_MODE.DOWNLOADING) {
        return d.status === DOWNLOAD_STATUS.DOWNLOADING || d.status === DOWNLOAD_STATUS.QUEUED;
      }
      if (filter === FILTER_MODE.COMPLETED) return d.status === DOWNLOAD_STATUS.COMPLETED;
      if (filter === FILTER_MODE.PAUSED) return d.status === DOWNLOAD_STATUS.PAUSED;
      return true;
    })
    .filter(d => {
      const q = searchQuery.trim().toLowerCase();
      if (!q) return true;
      return (d.filename || '').toLowerCase().includes(q) || (d.url || '').toLowerCase().includes(q);
    });
}

export function updateSelection({
  selectedIds,
  lastSelectedId,
  targetId,
  shiftKey = false,
  modifierKey = false,
  isCheckbox = false,
  visibleList = []
}) {
  let next = new Set(selectedIds);

  // 1. Shift + click range against visible list
  if (shiftKey && lastSelectedId && visibleList.length > 0) {
    const idx1 = visibleList.findIndex(d => d.id === lastSelectedId);
    const idx2 = visibleList.findIndex(d => d.id === targetId);
    if (idx1 !== -1 && idx2 !== -1) {
      if (!modifierKey) {
        next.clear();
      }
      const start = Math.min(idx1, idx2);
      const end = Math.max(idx1, idx2);
      for (let i = start; i <= end; i++) {
        next.add(visibleList[i].id);
      }
      return { nextSelectedIds: next, nextLastId: targetId };
    }
  }

  // 2. Explicit toggle via Checkbox or Cmd/Ctrl
  if (isCheckbox || modifierKey) {
    if (next.has(targetId)) {
      next.delete(targetId);
      return {
        nextSelectedIds: next,
        nextLastId: next.size === 0 ? null : (lastSelectedId === targetId ? null : lastSelectedId)
      };
    } else {
      next.add(targetId);
      return { nextSelectedIds: next, nextLastId: targetId };
    }
  }

  // 3. Normal Row Click
  if (selectedIds.size > 1) {
    // Sticky mode: toggle clicked item
    if (next.has(targetId)) {
      next.delete(targetId);
    } else {
      next.add(targetId);
    }
    return {
      nextSelectedIds: next,
      nextLastId: targetId
    };
  } else {
    // Single select mode: exclusive selection
    return {
      nextSelectedIds: new Set([targetId]),
      nextLastId: targetId
    };
  }
}
