import { useCallback } from 'react'
import { Button, Select, Space, Tooltip, App } from 'antd'
import {
  PlayCircleOutlined,
  ClearOutlined,
  FileTextOutlined,
  DownloadOutlined,
  CopyOutlined,
  FormatPainterOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import { buildMarkdownReport } from '@/api/sql'
import { formatSQL } from '@/utils/format'
import type { Dialect, LogType } from '@/types/sql'

const EXAMPLE_SQL = `SELECT
  u.id,
  u.name AS username,
  IFNULL(o.amount, 0) AS order_amount,
  DATE_FORMAT(o.created_at, '%Y-%m-%d') AS order_date
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
LEFT JOIN pay_record p ON p.order_id = o.id
WHERE u.status = 1
  AND (o.pay_type = 1 OR o.pay_type = 2)
  AND o.created_at BETWEEN '2026-01-01' AND '2026-01-31'
GROUP BY u.id, u.name, o.amount, o.created_at
ORDER BY o.created_at DESC
LIMIT 20;`

export default function SqlToolbar() {
  const { message } = App.useApp()
  const dialect = useSqlStore((s) => s.dialect)
  const logType = useSqlStore((s) => s.logType)
  const setDialect = useSqlStore((s) => s.setDialect)
  const setLogType = useSqlStore((s) => s.setLogType)
  const parseSql = useSqlStore((s) => s.parseSql)
  const clear = useSqlStore((s) => s.clear)
  const setRawText = useSqlStore((s) => s.setRawText)
  const result = useSqlStore((s) => s.result)
  const loading = useSqlStore((s) => s.loading)

  const loadExample = useCallback(() => {
    setRawText(EXAMPLE_SQL)
    message.success('已填入复杂示例 SQL')
  }, [setRawText, message])

  const handleFormat = useCallback(() => {
    const rawText = useSqlStore.getState().rawText
    if (!rawText.trim()) {
      message.warning('请先输入 SQL')
      return
    }
    const formatted = formatSQL(rawText, dialect)
    setRawText(formatted)
    message.success('SQL 已格式化')
  }, [dialect, setRawText, message])

  const handleExportMarkdown = useCallback(async () => {
    if (!result) return
    try {
      const res = await buildMarkdownReport(result)
      if (res.success && res.data) {
        const blob = new Blob([res.data.markdown], { type: 'text/markdown;charset=utf-8' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = 'sql-analysis-report.md'
        a.click()
        URL.revokeObjectURL(url)
        message.success('Markdown 报告已下载')
      }
    } catch {
      message.error('导出失败')
    }
  }, [result, message])

  const handleCopyReport = useCallback(async () => {
    if (!result) return
    try {
      const res = await buildMarkdownReport(result)
      if (res.success && res.data) {
        await navigator.clipboard.writeText(res.data.markdown)
        message.success('报告已复制')
      }
    } catch {
      message.error('复制失败，请重试')
    }
  }, [result, message])

  return (
    <div className="sql-lens-toolbar">
      <div className="sql-lens-toolbar-left">
        <Select<Dialect>
          value={dialect}
          onChange={(v) => setDialect(v)}
          size="small"
          style={{ width: 130, borderRadius: 6 }}
          options={[
            { value: 'mysql', label: '🐬 MySQL' },
            { value: 'postgresql', label: '🐘 PostgreSQL' },
            { value: 'oracle', label: '🔶 Oracle' },
            { value: 'sqlserver', label: '🏢 SQL Server' },
            { value: 'sqlite', label: '🪶 SQLite' },
          ]}
        />
        <Select<LogType>
          value={logType}
          onChange={(v) => setLogType(v)}
          size="small"
          style={{ width: 120, borderRadius: 6 }}
          options={[
            { value: 'auto', label: '🤖 自动识别' },
            { value: 'plain', label: '📝 原生 SQL' },
            { value: 'mybatis', label: '☕ MyBatis' },
            { value: 'laravel', label: '🟠 Laravel' },
            { value: 'thinkphp', label: '🐘 ThinkPHP' },
          ]}
        />
      </div>

      <div className="sql-lens-toolbar-right">
        <Space size={6}>
          <Tooltip title="填入复杂示例 SQL（含 IFNULL/DATE_FORMAT）">
            <Button size="small" icon={<FileTextOutlined />} onClick={loadExample}>
              示例
            </Button>
          </Tooltip>
          <Tooltip title="格式化 SQL (Ctrl+Shift+F)">
            <Button size="small" icon={<FormatPainterOutlined />} onClick={handleFormat}>
              格式化
            </Button>
          </Tooltip>
          <Tooltip title="复制解析报告">
            <Button size="small" icon={<CopyOutlined />} onClick={handleCopyReport} disabled={!result} />
          </Tooltip>
          <Tooltip title="下载 Markdown 报告">
            <Button size="small" icon={<DownloadOutlined />} onClick={handleExportMarkdown} disabled={!result} />
          </Tooltip>
          <Button size="small" icon={<ClearOutlined />} onClick={clear} disabled={loading}>
            清空
          </Button>
          <Button
            size="small"
            type="primary"
            icon={<PlayCircleOutlined />}
            onClick={parseSql}
            loading={loading}
            style={{ fontWeight: 600 }}
          >
            解析
          </Button>
        </Space>
      </div>
    </div>
  )
}
