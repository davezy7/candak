import { writable } from 'svelte/store';

const STORAGE_KEY = 'candak_theme';

function getInitialTheme() {
  if (typeof window === 'undefined') return 'system';
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === 'light' || saved === 'dark' || saved === 'system') {
    return saved;
  }
  return 'system';
}

export const theme = writable(getInitialTheme());

export function initTheme() {
  if (typeof window === 'undefined') return;

  const media = window.matchMedia('(prefers-color-scheme: dark)');

  function apply(t) {
    const isDark = t === 'dark' || (t === 'system' && media.matches);
    if (isDark) {
      document.documentElement.setAttribute('data-theme', 'dark');
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.setAttribute('data-theme', 'light');
      document.documentElement.classList.remove('dark');
    }
  }

  theme.subscribe((val) => {
    localStorage.setItem(STORAGE_KEY, val);
    apply(val);
  });

  media.addEventListener('change', () => {
    let current = getInitialTheme();
    if (current === 'system') {
      apply('system');
    }
  });
}
