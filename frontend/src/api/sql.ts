import axios from 'axios'
import type { ParseSQLRequest, SQLAnalysisResult, APIResponse } from '@/types/sql'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// Response interceptor for better error messages
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const data = error.response.data
      // Try to extract meaningful error from JSON response
      if (data && typeof data === 'object' && data.error) {
        return Promise.reject(new Error(data.error.message || '请求失败'))
      }
      // HTML error pages
      if (typeof data === 'string' && data.includes('<!DOCTYPE')) {
        return Promise.reject(new Error(`服务器错误 (${error.response.status})`))
      }
      return Promise.reject(new Error(`请求失败 (${error.response.status})`))
    }
    if (error.code === 'ECONNABORTED') {
      return Promise.reject(new Error('请求超时，请检查 SQL 是否过长'))
    }
    return Promise.reject(new Error('网络连接失败，请检查后端服务是否启动'))
  }
)

export async function parseSql(req: ParseSQLRequest): Promise<APIResponse<SQLAnalysisResult>> {
  const { data } = await api.post<APIResponse<SQLAnalysisResult>>('/sql/parse', req)
  return data
}

export async function formatSql(sql: string, dialect: string): Promise<APIResponse<{ formattedSql: string }>> {
  const { data } = await api.post('/sql/format', { sql, dialect })
  return data
}

export async function extractSql(rawLog: string, logType: string): Promise<APIResponse<{
  sql: string
  bindings: unknown[]
  restoredSql: string
  logType: string
}>> {
  const { data } = await api.post('/log/extract-sql', { rawLog, logType })
  return data
}

export async function buildMarkdownReport(analysisResult: SQLAnalysisResult): Promise<APIResponse<{ markdown: string }>> {
  const { data } = await api.post('/report/markdown', { analysisResult })
  return data
}
