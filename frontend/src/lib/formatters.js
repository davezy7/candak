export function formatBytes(bytes, decimals = 1) {
  if (bytes === null || bytes === undefined || isNaN(bytes)) return '0 B';
  if (bytes === 0) return '0 B';
  if (bytes < 0) return 'Unknown';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));
  if (i < 0) return bytes + ' B';
  const sizeIdx = Math.min(i, sizes.length - 1);

  return parseFloat((bytes / Math.pow(k, sizeIdx)).toFixed(dm)) + ' ' + sizes[sizeIdx];
}

export function formatSpeed(bytesPerSec) {
  if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
  return formatBytes(bytesPerSec, 1) + '/s';
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

export function formatDate(dateString) {
  if (!dateString) return '';
  try {
    const d = new Date(dateString);
    return d.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  } catch (e) {
    return dateString;
  }
}

export function getFileCategory(filename) {
  if (!filename) return 'other';
  const ext = filename.split('.').pop()?.toLowerCase();
  
  const categories = {
    archive: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'iso', 'dmg', 'pkg'],
    audio: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a'],
    video: ['mp4', 'mkv', 'avi', 'mov', 'webm', 'wmv'],
    image: ['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'avif', 'bmp'],
    document: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md'],
    code: ['go', 'js', 'ts', 'py', 'html', 'css', 'json', 'yaml', 'yml', 'rs', 'c', 'cpp', 'sh'],
    executable: ['exe', 'app', 'msi', 'bin', 'apk']
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
