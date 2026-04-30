import { useCallback } from 'react'
import { Spin, Alert } from 'antd'
import { useSqlStore } from '@/store/useSqlStore'
import SqlToolbar from '@/components/SqlToolbar/SqlToolbar'
import SqlEditor from '@/components/SqlEditor/SqlEditor'
import SummaryCards from '@/components/SummaryCards/SummaryCards'
import ResultTabs from '@/components/ResultTabs/ResultTabs'
import EmptyState from '@/components/EmptyState/EmptyState'

export default function Home() {
  const loading = useSqlStore((s) => s.loading)
  const error = useSqlStore((s) => s.error)
  const result = useSqlStore((s) => s.result)
  const clear = useSqlStore((s) => s.clear)

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      useSqlStore.getState().parseSql()
    }
  }, [])

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }} onKeyDown={handleKeyDown}>
      {/* Header */}
      <header className="sql-lens-header">
        <div className="sql-lens-header-logo">
          <svg viewBox="0 0 32 32" fill="none">
            <rect width="32" height="32" rx="6" fill="#1677ff" />
            <path d="M9 11h14M9 16h10M9 21h6" stroke="white" strokeWidth="2.5" strokeLinecap="round" />
            <circle cx="24" cy="21" r="3" fill="#52c41a" />
          </svg>
          <span className="sql-lens-header-title">SQL Lens</span>
          <span className="sql-lens-header-badge">v0.1</span>
        </div>
      </header>

      {/* Toolbar */}
      <SqlToolbar />

      {/* Main Content */}
      <div className="sql-lens-content">
        {/* Left Panel */}
        <div className="sql-lens-left-panel">
          <div className="sql-lens-editor-wrapper">
            <SqlEditor />
          </div>
          <div className="sql-lens-status-bar">
            {loading ? (
              <>
                <Spin size="small" />
                <span>正在解析 SQL...</span>
              </>
            ) : error ? (
              <span className="error-text">解析错误</span>
            ) : result ? (
              <>
                <span>解析完成</span>
                <span>{result.summary.tableCount} 表</span>
                <span>{result.summary.joinCount} JOIN</span>
                <span>{result.summary.fieldCount} 字段</span>
              </>
            ) : (
              <span>Ctrl+Enter 解析 SQL</span>
            )}
          </div>
        </div>

        {/* Right Panel */}
        <div className="sql-lens-right-panel">
          {error && (
            <Alert
              message="解析失败"
              description={error}
              type="error"
              closable
              showIcon
              style={{ margin: '12px 16px 0', borderRadius: 8 }}
              onClose={() => clear()}
            />
          )}
          {result ? (
            <>
              <SummaryCards />
              <div className="sql-lens-tabs-wrapper">
                <ResultTabs />
              </div>
            </>
          ) : (
            <EmptyState />
          )}
        </div>
      </div>
    </div>
  )
}
