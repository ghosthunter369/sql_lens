import { useState } from 'react'
import { Button, Empty, Tag, Divider } from 'antd'
import { SwapOutlined, TableOutlined, LinkOutlined, FieldStringOutlined } from '@ant-design/icons'
import Editor from '@monaco-editor/react'
import { useSqlStore } from '@/store/useSqlStore'
import { parseSql } from '@/api/sql'
import type { SQLAnalysisResult } from '@/types/sql'

export default function DiffPanel() {
  const result = useSqlStore((s) => s.result)
  const rawText = useSqlStore((s) => s.rawText)
  const [compareSql, setCompareSql] = useState('')
  const [compareResult, setCompareResult] = useState<SQLAnalysisResult | null>(null)
  const [loading, setLoading] = useState(false)

  const handleCompare = async () => {
    if (!compareSql.trim()) return
    setLoading(true)
    try {
      const dialect = useSqlStore.getState().dialect
      const response = await parseSql({
        rawText: compareSql,
        dialect,
        logType: 'auto',
        options: { restoreBindings: true, formatSql: true, enableRiskCheck: true },
      })
      if (response.success && response.data) {
        setCompareResult(response.data)
      }
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  if (!result) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}>
        <Empty description="请先解析一条 SQL 语句" />
      </div>
    )
  }

  return (
    <div style={{ padding: 12, height: '100%', display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <SwapOutlined style={{ color: '#13c2c2' }} />
        <span style={{ fontWeight: 600, fontSize: 14 }}>SQL 对比</span>
      </div>

      <div style={{ display: 'flex', gap: 12, flex: 1, minHeight: 0 }}>
        {/* Left: Current */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>当前 SQL</div>
          <div style={{ flex: 1, border: '1px solid #f0f0f0', borderRadius: 6, overflow: 'hidden' }}>
            <Editor
              height="100%"
              language="sql"
              value={rawText}
              theme="vs-dark"
              options={{ readOnly: true, minimap: { enabled: false }, fontSize: 12, lineNumbers: 'on' }}
            />
          </div>
        </div>

        {/* Right: Compare */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>对比 SQL</div>
          <div style={{ flex: 1, border: '1px solid #f0f0f0', borderRadius: 6, overflow: 'hidden' }}>
            <Editor
              height="100%"
              language="sql"
              value={compareSql}
              onChange={(v) => setCompareSql(v || '')}
              theme="vs-dark"
              options={{ minimap: { enabled: false }, fontSize: 12, lineNumbers: 'on' }}
            />
          </div>
        </div>
      </div>

      <Button
        type="primary"
        icon={<SwapOutlined />}
        onClick={handleCompare}
        loading={loading}
        disabled={!compareSql.trim()}
        style={{ alignSelf: 'center' }}
      >
        对比分析
      </Button>

      {compareResult && (
        <DiffSummary left={result} right={compareResult} />
      )}
    </div>
  )
}

function DiffSummary({ left, right }: { left: SQLAnalysisResult; right: SQLAnalysisResult }) {
  const diff = (a: number, b: number) => {
    if (a === b) return <Tag color="default">{a}</Tag>
    return a > b
      ? <Tag color="red">+{a - b} ({a})</Tag>
      : <Tag color="green">-{b - a} ({a})</Tag>
  }

  const strDiff = (a: string, b: string) => {
    if (a === b) return <Tag color="default">{a}</Tag>
    return <Tag color="orange">{a} → {b}</Tag>
  }

  return (
    <div style={{ background: '#fafafa', borderRadius: 8, padding: 12, border: '1px solid #f0f0f0' }}>
      <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8 }}>对比结果</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, fontSize: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <TableOutlined /> 表数量: {diff(left.summary.tableCount, right.summary.tableCount)}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <LinkOutlined /> JOIN 数: {diff(left.summary.joinCount, right.summary.joinCount)}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <FieldStringOutlined /> 字段数: {diff(left.summary.fieldCount, right.summary.fieldCount)}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          WHERE 条件: {diff(left.summary.whereCount, right.summary.whereCount)}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          复杂度: {strDiff(left.summary.complexity, right.summary.complexity)}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          风险数: {diff(left.risks.length, right.risks.length)}
        </div>
      </div>

      {/* Table diff */}
      <Divider style={{ margin: '8px 0' }} />
      <div style={{ fontSize: 12 }}>
        <div style={{ fontWeight: 600, marginBottom: 4 }}>表差异</div>
        {(() => {
          const leftTables = new Set(left.tables.map((t) => t.name))
          const rightTables = new Set(right.tables.map((t) => t.name))
          const onlyLeft = left.tables.filter((t) => !rightTables.has(t.name))
          const onlyRight = right.tables.filter((t) => !leftTables.has(t.name))
          const common = left.tables.filter((t) => rightTables.has(t.name))
          return (
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              {common.map((t) => <Tag key={t.name} color="default">{t.name}</Tag>)}
              {onlyLeft.map((t) => <Tag key={t.name} color="red">仅左: {t.name}</Tag>)}
              {onlyRight.map((t) => <Tag key={t.name} color="green">仅右: {t.name}</Tag>)}
            </div>
          )
        })()}
      </div>
    </div>
  )
}
