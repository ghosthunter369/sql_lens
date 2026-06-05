import { create } from 'zustand'
import type { SQLAnalysisResult } from '@/types/sql'

export interface HistoryItem {
  id: string
  sql: string
  dialect: string
  timestamp: number
  summary: {
    tableCount: number
    joinCount: number
    fieldCount: number
    complexity: string
  }
  starred: boolean
}

const STORAGE_KEY = 'sql-lens-history'
const MAX_HISTORY = 100

function loadHistory(): HistoryItem[] {
  try {
    const data = localStorage.getItem(STORAGE_KEY)
    return data ? JSON.parse(data) : []
  } catch {
    return []
  }
}

function saveHistory(items: HistoryItem[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items.slice(0, MAX_HISTORY)))
  } catch {
    // localStorage full, ignore
  }
}

export interface HistoryState {
  history: HistoryItem[]
  addHistory: (sql: string, dialect: string, result: SQLAnalysisResult) => void
  removeHistory: (id: string) => void
  toggleStar: (id: string) => void
  clearHistory: () => void
  loadSql: (id: string) => string | null
}

export const useHistoryStore = create<HistoryState>((set, get) => ({
  history: loadHistory(),

  addHistory: (sql: string, dialect: string, result: SQLAnalysisResult) => {
    const trimmed = sql.trim()
    if (!trimmed) return

    const item: HistoryItem = {
      id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
      sql: trimmed.length > 2000 ? trimmed.slice(0, 2000) + '...' : trimmed,
      dialect,
      timestamp: Date.now(),
      summary: {
        tableCount: result.summary.tableCount,
        joinCount: result.summary.joinCount,
        fieldCount: result.summary.fieldCount,
        complexity: result.summary.complexity,
      },
      starred: false,
    }

    // Deduplicate by SQL content
    const existing = get().history.filter((h) => h.sql !== item.sql)
    const newHistory = [item, ...existing].slice(0, MAX_HISTORY)
    set({ history: newHistory })
    saveHistory(newHistory)
  },

  removeHistory: (id: string) => {
    const newHistory = get().history.filter((h) => h.id !== id)
    set({ history: newHistory })
    saveHistory(newHistory)
  },

  toggleStar: (id: string) => {
    const newHistory = get().history.map((h) =>
      h.id === id ? { ...h, starred: !h.starred } : h
    )
    set({ history: newHistory })
    saveHistory(newHistory)
  },

  clearHistory: () => {
    set({ history: [] })
    saveHistory([])
  },

  loadSql: (id: string) => {
    const item = get().history.find((h) => h.id === id)
    return item ? item.sql : null
  },
}))
