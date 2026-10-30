export function formatBytes(bytes, decimals = 1, locale) {
  if (bytes === null || bytes === undefined || isNaN(bytes)) return '0 B';
  if (bytes === 0) return '0 B';
  if (bytes < 0) return 'Unknown';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));
  if (i < 0) return bytes + ' B';
  const sizeIdx = Math.min(i, sizes.length - 1);
  const val = bytes / Math.pow(k, sizeIdx);

  const formatted = new Intl.NumberFormat(locale || 'en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits: dm
  }).format(val);

  return formatted + ' ' + sizes[sizeIdx];
}

export function formatSpeed(bytesPerSec, locale) {
  if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
  return formatBytes(bytesPerSec, 1, locale) + '/s';
}

export function formatETA(seconds) {
  if (seconds === null || seconds === undefined || seconds <= 0 || !isFinite(seconds)) return '';
  const totalSecs = Math.floor(seconds);
  const mins = Math.floor(totalSecs / 60);
  const secs = totalSecs % 60;
  const pad = (n) => n.toString().padStart(2, '0');

  if (totalSecs < 3600) {
    return `${pad(mins)}:${pad(secs)} remaining`;
  }
  const hours = Math.floor(totalSecs / 3600);
  const remMins = Math.floor((totalSecs % 3600) / 60);
  return `${pad(hours)}:${pad(remMins)}:${pad(secs)} remaining`;
}

export function formatDate(dateString, locale) {
  if (!dateString) return '';
  try {
    const d = new Date(dateString);
    return new Intl.DateTimeFormat(locale || undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(d);
  } catch (e) {
    return dateString;
  }
}

export function getFileCategory(filename, savePath) {
  const target = filename || savePath || '';
  const ext = getFileExtension(target);
  if (!ext) return 'other';
  
  const categories = {
    archive: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'iso', 'dmg', 'pkg'],
    audio: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a', 'opus'],
    video: ['mp4', 'mkv', 'avi', 'mov', 'webm', 'wmv', 'm4v', 'flv'],
    image: ['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'avif', 'bmp', 'ico', 'tiff'],
    document: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'epub', 'rtf'],
    code: ['go', 'js', 'ts', 'py', 'html', 'css', 'json', 'yaml', 'yml', 'rs', 'c', 'cpp', 'sh', 'sql'],
    executable: ['exe', 'app', 'msi', 'bin', 'apk', 'deb', 'rpm']
  };

  for (const [category, exts] of Object.entries(categories)) {
    if (exts.includes(ext)) return category;
  }
  return 'other';
}

export const DEFAULT_ACCEPTED_EXTENSIONS = [
  'zip', 'rar', '7z', 'tar', 'gz', 'iso', 'dmg', 'pkg',
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'txt',
  'mp4', 'mkv', 'mov', 'mp3', 'wav', 'flac',
  'jpg', 'jpeg', 'png', 'webp', 'gif', 'svg',
  'exe', 'msi', 'apk', 'bin'
];

export function getFileExtension(filename) {
  if (!filename) return '';
  const clean = filename.split('?')[0].split('#')[0];
  const basename = clean.split('/').pop();
  if (!basename) return '';
  const parts = basename.split('.');
  if (parts.length <= 1) return '';
  return parts.pop().toLowerCase().trim();
}

export function getExtensionBadge(filename, savePath) {
  const target = (filename && filename.includes('.')) ? filename : (savePath || filename || '');
  const basename = target.split(/[/\\]/).pop() || '';
  if (!basename || !basename.includes('.')) return 'FILE';
  const ext = basename.split('.').pop()?.trim().toUpperCase() || 'FILE';
  return ext.length > 5 ? ext.slice(0, 4) : ext;
}

export function getFolderPath(savePath) {
  if (!savePath) return '';
  const separator = savePath.includes('\\') ? '\\' : '/';
  if (!savePath.includes(separator)) return '';
  const parts = savePath.split(separator);
  parts.pop();
  const folder = parts.join(separator);
  if (!folder) return separator;
  if (folder.endsWith(':')) return folder + separator;
  return folder;
}

