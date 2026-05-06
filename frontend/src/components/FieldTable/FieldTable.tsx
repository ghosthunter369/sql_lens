import { Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useSqlStore } from '@/store/useSqlStore'
import type { FieldMeta, DeepSourceRef, WindowSpecMeta, FuncCategory } from '@/types/sql'

const fieldTypeColors: Record<string, { color: string; bg: string; border: string }> = {
  column:    { color: '#1677ff', bg: '#e6f4ff', border: '#91caff' },
  function:  { color: '#722ed1', bg: '#f9f0ff', border: '#d3adf7' },
  aggregate: { color: '#fa8c16', bg: '#fff7e6', border: '#ffd591' },
  case:      { color: '#ff4d4f', bg: '#fff2f0', border: '#ffccc7' },
  subquery:  { color: '#13c2c2', bg: '#e6fffb', border: '#87e8de' },
  wildcard:  { color: '#8c8c8c', bg: '#fafafa', border: '#d9d9d9' },
  window:    { color: '#eb2f96', bg: '#fff0f6', border: '#ffadd2' },
}

const fieldTypeLabels: Record<string, string> = {
  column: '普通字段',
  function: '函数',
  aggregate: '聚合',
  case: 'CASE',
  subquery: '子查询',
  wildcard: '通配',
  window: '窗口函数',
}

const funcCategoryLabels: Record<FuncCategory, string> = {
  aggregate: '聚合',
  window: '窗口',
  scalar: '标量',
  datetime: '日期时间',
  string: '字符串',
  math: '数学',
  conditional: '条件',
  cast: '类型转换',
  json: 'JSON',
}

const funcCategoryColors: Record<FuncCategory, string> = {
  aggregate: '#fa8c16',
  window: '#eb2f96',
  scalar: '#722ed1',
  datetime: '#13c2c2',
  string: '#52c41a',
  math: '#1677ff',
  conditional: '#ff4d4f',
  cast: '#faad14',
  json: '#2f54eb',
}

function DeepSourcesCell({ sources }: { sources?: DeepSourceRef[] }) {
  if (!sources || sources.length === 0) {
    return <span style={{ color: '#d9d9d9' }}>—</span>
  }
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 3 }}>
      {sources.map((ds, i) => (
        <Tooltip key={i} title={`${ds.table}.${ds.column}${ds.alias ? ` (别名: ${ds.alias})` : ''}`}>
          <Tag style={{ margin: 0, fontFamily: 'var(--font-mono)', fontSize: 10, lineHeight: '18px', padding: '0 4px' }}>
            {ds.table}.{ds.column}
          </Tag>
        </Tooltip>
      ))}
    </div>
  )
}

function WindowSpecCell({ spec }: { spec?: WindowSpecMeta }) {
  if (!spec) return <span style={{ color: '#d9d9d9' }}>—</span>
  const parts: string[] = []
  if (spec.partitionBy && spec.partitionBy.length > 0) {
    parts.push(`PARTITION BY ${spec.partitionBy.join(', ')}`)
  }
  if (spec.orderBy && spec.orderBy.length > 0) {
    parts.push(`ORDER BY ${spec.orderBy.map(o => `${o.expression} ${o.direction}`).join(', ')}`)
  }
  if (spec.frameClause) {
    parts.push(spec.frameClause)
  }
  return (
    <Tooltip title={parts.join(' ')} placement="topLeft">
      <div style={{ fontSize: 11, maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {parts.map((p, i) => (
          <div key={i} style={{ color: '#666' }}>{p}</div>
        ))}
      </div>
    </Tooltip>
  )
}

const columns: ColumnsType<FieldMeta> = [
  {
    title: '输出字段',
    dataIndex: 'outputName',
    key: 'outputName',
    width: 140,
    ellipsis: true,
    render: (v: string) => (
      <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>{v}</span>
    ),
  },
  {
    title: '浅层来源',
    key: 'shallowSource',
    width: 140,
    render: (_: unknown, record: FieldMeta) => (
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {record.sourceTable ? (
          <Tag style={{ margin: 0, fontFamily: 'var(--font-mono)', fontSize: 11 }}>{record.sourceTable}</Tag>
        ) : <span style={{ color: '#d9d9d9' }}>—</span>}
        {record.sourceColumn && (
          <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: '#434343' }}>.{record.sourceColumn}</code>
        )}
      </div>
    ),
  },
  {
    title: '深度来源',
    key: 'deepSources',
    width: 180,
    render: (_: unknown, record: FieldMeta) => <DeepSourcesCell sources={record.deepSources} />,
  },
  {
    title: '原始表达式',
    dataIndex: 'expression',
    key: 'expression',
    ellipsis: true,
    render: (v: string) => (
      <Tooltip title={v} placement="topLeft">
        <code className="field-expr-code">{v}</code>
      </Tooltip>
    ),
  },
  {
    title: '类型',
    dataIndex: 'fieldType',
    key: 'fieldType',
    width: 80,
    render: (v: string) => {
      const c = fieldTypeColors[v] || fieldTypeColors.wildcard
      return (
        <Tag style={{
          color: c.color,
          background: c.bg,
          border: `1px solid ${c.border}`,
          margin: 0,
          fontWeight: 500,
        }}>
          {fieldTypeLabels[v] || v}
        </Tag>
      )
    },
  },
  {
    title: '函数分类',
    dataIndex: 'funcCategory',
    key: 'funcCategory',
    width: 90,
    render: (v?: FuncCategory) => {
      if (!v) return <span style={{ color: '#d9d9d9' }}>—</span>
      return (
        <Tag style={{
          color: funcCategoryColors[v] || '#666',
          background: '#fafafa',
          border: `1px solid ${funcCategoryColors[v] || '#d9d9d9'}`,
          margin: 0,
          fontSize: 11,
        }}>
          {funcCategoryLabels[v] || v}
        </Tag>
      )
    },
  },
  {
    title: '窗口规格',
    key: 'windowSpec',
    width: 200,
    render: (_: unknown, record: FieldMeta) => <WindowSpecCell spec={record.windowSpec} />,
  },
]

export default function FieldTable() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  if (result.fields.length === 0) {
    return (
      <div className="tab-empty-state">
        <span>暂无查询字段</span>
      </div>
    )
  }

  return (
    <div className="field-table-wrapper">
      <div className="tab-section-header">
        共 <strong>{result.fields.length}</strong> 个字段
      </div>
      <Table<FieldMeta>
        columns={columns}
        dataSource={result.fields}
        rowKey="id"
        size="small"
        pagination={false}
        scroll={{ y: 420 }}
        style={{ fontSize: 13 }}
        rowClassName={() => 'field-table-row'}
      />
    </div>
  )
}
