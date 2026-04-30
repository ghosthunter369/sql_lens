import axios from 'axios'
import type { ParseSQLRequest, SQLAnalysisResult, APIResponse } from '@/types/sql'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

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
