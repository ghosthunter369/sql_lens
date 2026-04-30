import { create } from 'zustand'
import type { Dialect, LogType, SQLAnalysisResult } from '@/types/sql'
import { parseSql as apiParseSql } from '@/api/sql'

export interface SqlState {
  rawText: string
  dialect: Dialect
  logType: LogType
  loading: boolean
  result?: SQLAnalysisResult
  error?: string

  setRawText: (text: string) => void
  setDialect: (dialect: Dialect) => void
  setLogType: (logType: LogType) => void
  parseSql: () => Promise<void>
  setResult: (result: SQLAnalysisResult | undefined) => void
  clear: () => void
}

export const useSqlStore = create<SqlState>((set, get) => ({
  rawText: '',
  dialect: 'mysql',
  logType: 'auto',
  loading: false,
  result: undefined,
  error: undefined,

  setRawText: (text: string) => set({ rawText: text, error: undefined }),

  setDialect: (dialect: Dialect) => set({ dialect }),

  setLogType: (logType: LogType) => set({ logType }),

  parseSql: async () => {
    const { rawText, dialect, logType } = get()
    if (!rawText.trim()) {
      set({ error: '请输入 SQL 或日志文本' })
      return
    }

    set({ loading: true, error: undefined })

    try {
      const response = await apiParseSql({
        rawText,
        dialect,
        logType,
        options: {
          restoreBindings: true,
          formatSql: true,
          enableRiskCheck: true,
        },
      })

      if (response.success && response.data) {
        set({ result: response.data, loading: false })
      } else if (response.error) {
        set({ error: response.error.message, loading: false })
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : '解析请求失败', loading: false })
    }
  },

  setResult: (result) => set({ result }),

  clear: () =>
    set({
      rawText: '',
      result: undefined,
      error: undefined,
    }),
}))
