import { useState, useCallback } from 'react'
import { Spin, Alert, Button, Tooltip, Drawer, Segmented } from 'antd'
import {
  HistoryOutlined,
  BookOutlined,
  SwapOutlined,
  ThunderboltOutlined,
  BulbOutlined,
  BulbFilled,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import { useThemeStore } from '@/store/useThemeStore'
import { useHistoryStore } from '@/store/useHistoryStore'
import { formatSQL } from '@/utils/format'
import SqlToolbar from '@/components/SqlToolbar/SqlToolbar'
import SqlEditor from '@/components/SqlEditor/SqlEditor'
import SummaryCards from '@/components/SummaryCards/SummaryCards'
import ResultTabs from '@/components/ResultTabs/ResultTabs'
import EmptyState from '@/components/EmptyState/EmptyState'
import HistoryPanel from '@/components/HistoryPanel/HistoryPanel'
import SnippetPanel from '@/components/SnippetPanel/SnippetPanel'
import DiffPanel from '@/components/DiffPanel/DiffPanel'
import PerformancePanel from '@/components/PerformancePanel/PerformancePanel'
import ExportMenu from '@/components/ExportMenu/ExportMenu'

type SidePanel = 'history' | 'snippets' | 'diff' | 'perf' | null

export default function Home() {
  const loading = useSqlStore((s) => s.loading)
  const error = useSqlStore((s) => s.error)
  const result = useSqlStore((s) => s.result)
  const clearError = useSqlStore((s) => s.clearError)
  const setRawText = useSqlStore((s) => s.setRawText)
  const dialect = useSqlStore((s) => s.dialect)
  const addHistory = useHistoryStore((s) => s.addHistory)
  const { theme, resolvedTheme, toggleTheme } = useThemeStore()
  const [sidePanel, setSidePanel] = useState<SidePanel>(null)

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      useSqlStore.getState().parseSql()
    }
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
      e.preventDefault()
      const rawText = useSqlStore.getState().rawText
      if (rawText.trim()) {
        const formatted = formatSQL(rawText, dialect)
        setRawText(formatted)
      }
    }
  }, [setRawText, dialect])

  // Save to history when parse succeeds
  const prevResult = useState(() => ({ current: null as typeof result | null }))[0]
  if (result && result !== prevResult.current) {
    prevResult.current = result
    const rawText = useSqlStore.getState().rawText
    if (rawText.trim()) {
      addHistory(rawText, dialect, result)
    }
  }

  const sidePanelItems = [
    { key: 'history', icon: <HistoryOutlined />, label: '历史' },
    { key: 'snippets', icon: <BookOutlined />, label: '片段' },
    { key: 'diff', icon: <SwapOutlined />, label: '对比' },
    { key: 'perf', icon: <ThunderboltOutlined />, label: '性能' },
  ]

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
          <span className="sql-lens-header-badge">v0.3</span>
        </div>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          {result && <ExportMenu />}
          {sidePanelItems.map((item) => (
            <Tooltip key={item.key} title={item.label}>
              <Button
                type={sidePanel === item.key ? 'primary' : 'text'}
                size="small"
                icon={item.icon}
                onClick={() => setSidePanel(sidePanel === item.key ? null : item.key as SidePanel)}
                style={{ color: sidePanel === item.key ? undefined : '#9ca3af' }}
              />
            </Tooltip>
          ))}
          <Tooltip title={resolvedTheme === 'dark' ? '切换亮色模式' : '切换暗色模式'}>
            <Button
              type="text"
              size="small"
              icon={resolvedTheme === 'dark' ? <BulbFilled /> : <BulbOutlined />}
              onClick={toggleTheme}
              style={{ color: '#9ca3af' }}
            />
          </Tooltip>
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
              onClose={() => clearError()}
            />
          )}
          {loading ? (
            <div className="sql-lens-loading">
              <Spin size="large" />
              <span>正在解析中...</span>
            </div>
          ) : result ? (
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

        {/* Side Panel Drawer */}
        {sidePanel && (
          <Drawer
            title={
              sidePanelItems.find((i) => i.key === sidePanel)?.label || ''
            }
            placement="right"
            open={!!sidePanel}
            onClose={() => setSidePanel(null)}
            width={380}
            styles={{ body: { padding: 0 } }}
            mask={false}
            getContainer={false}
            style={{ position: 'absolute' }}
          >
            {sidePanel === 'history' && <HistoryPanel />}
            {sidePanel === 'snippets' && <SnippetPanel />}
            {sidePanel === 'diff' && <DiffPanel />}
            {sidePanel === 'perf' && <PerformancePanel />}
          </Drawer>
        )}
      </div>
    </div>
  )
}
