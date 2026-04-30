import { format as sqlFormat } from 'sql-formatter'

export function formatSQL(sql: string, dialect: string = 'mysql'): string {
  try {
    const language = dialect === 'postgresql' ? 'postgresql' : 'mysql'
    return sqlFormat(sql, { language: language as 'mysql' | 'postgresql' })
  } catch {
    return sql
  }
}
