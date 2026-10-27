// app/theme/bootstrap.ts
interface Grounds {
  light: string
  dark: string
}

interface StorageKeys {
  settings: string
  accent: string
  paint: string
}

// Serialized into the document head so it runs before Nuxt mounts.
export function restoreTheme(grounds: Grounds, keys: StorageKeys) {
  let mode = 'auto'
  try {
    const prefix = `${keys.settings}=`
    const cookie = document.cookie.split('; ').find(value => value.startsWith(prefix))
    if (cookie) mode = JSON.parse(decodeURIComponent(cookie.slice(prefix.length))).themeMode
  }
  catch {
    // Invalid preferences use the system theme.
  }

  const dark = mode === 'dark' || (mode !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches)
  let colors = grounds
  try {
    const saved = JSON.parse(localStorage.getItem(keys.paint) || 'null')
    if (saved?.seed === localStorage.getItem(keys.accent)
      && /^#[0-9a-f]{6}$/i.test(saved.light)
      && /^#[0-9a-f]{6}$/i.test(saved.dark)) {
      colors = saved
    }
  }
  catch {
    // Paint the default background when storage is unavailable.
  }

  document.documentElement.style.backgroundColor = dark ? colors.dark : colors.light
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
}
