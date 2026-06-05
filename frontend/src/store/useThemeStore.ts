import { create } from 'zustand'

type Theme = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'sql-lens-theme'

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function loadTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY) as Theme
    if (stored === 'light' || stored === 'dark' || stored === 'auto') return stored
  } catch {}
  return 'auto'
}

function applyTheme(theme: Theme) {
  const resolved = theme === 'auto' ? getSystemTheme() : theme
  document.documentElement.setAttribute('data-theme', resolved)
}

export interface ThemeState {
  theme: Theme
  resolvedTheme: 'light' | 'dark'
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

export const useThemeStore = create<ThemeState>((set, get) => {
  const initial = loadTheme()
  // Apply on next tick to ensure DOM is ready
  setTimeout(() => applyTheme(initial), 0)

  // Listen for system theme changes
  if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (get().theme === 'auto') {
        applyTheme('auto')
        set({ resolvedTheme: getSystemTheme() })
      }
    })
  }

  return {
    theme: initial,
    resolvedTheme: initial === 'auto' ? getSystemTheme() : initial,

    setTheme: (theme: Theme) => {
      localStorage.setItem(STORAGE_KEY, theme)
      applyTheme(theme)
      const resolved = theme === 'auto' ? getSystemTheme() : theme
      set({ theme, resolvedTheme: resolved })
    },

    toggleTheme: () => {
      const current = get().resolvedTheme
      const next = current === 'dark' ? 'light' : 'dark'
      get().setTheme(next)
    },
  }
})
