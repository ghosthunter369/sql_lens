import { format as sqlFormat } from 'sql-formatter'

const dialectLanguageMap: Record<string, string> = {
  mysql: 'mysql',
  postgresql: 'postgresql',
  oracle: 'plsql',
  sqlserver: 'transactsql',
  sqlite: 'sqlite',
}

export function formatSQL(sql: string, dialect: string = 'mysql'): string {
  try {
    const language = dialectLanguageMap[dialect] || 'mysql'
    return sqlFormat(sql, { language: language as 'mysql' | 'postgresql' | 'plsql' | 'transactsql' | 'sqlite' })
  } catch {
    return sql
  }
}
