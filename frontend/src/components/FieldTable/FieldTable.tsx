import { Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useSqlStore } from '@/store/useSqlStore'
import type { FieldMeta } from '@/types/sql'

const fieldTypeColors: Record<string, { color: string; bg: string; border: string }> = {
  column:    { color: '#1677ff', bg: '#e6f4ff', border: '#91caff' },
  function:  { color: '#722ed1', bg: '#f9f0ff', border: '#d3adf7' },
  aggregate: { color: '#fa8c16', bg: '#fff7e6', border: '#ffd591' },
  case:      { color: '#ff4d4f', bg: '#fff2f0', border: '#ffccc7' },
  subquery:  { color: '#13c2c2', bg: '#e6fffb', border: '#87e8de' },
  wildcard:  { color: '#8c8c8c', bg: '#fafafa', border: '#d9d9d9' },
}

const fieldTypeLabels: Record<string, string> = {
  column: '普通字段',
  function: '函数',
  aggregate: '聚合',
  case: 'CASE',
  subquery: '子查询',
  wildcard: '通配',
}

const columns: ColumnsType<FieldMeta> = [
  {
    title: '输出字段',
    dataIndex: 'outputName',
    key: 'outputName',
    width: 150,
    ellipsis: true,
    render: (v: string) => (
      <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>{v}</span>
    ),
  },
  {
    title: '来源表',
    dataIndex: 'sourceTable',
    key: 'sourceTable',
    width: 130,
    render: (v: string) => v ? (
      <Tag style={{ margin: 0, fontFamily: 'var(--font-mono)', fontSize: 11 }}>{v}</Tag>
    ) : <span style={{ color: '#d9d9d9' }}>—</span>,
  },
  {
    title: '别名',
    dataIndex: 'sourceAlias',
    key: 'sourceAlias',
    width: 70,
    render: (v: string) => v || <span style={{ color: '#d9d9d9' }}>—</span>,
  },
  {
    title: '来源字段',
    dataIndex: 'sourceColumn',
    key: 'sourceColumn',
    width: 120,
    ellipsis: true,
    render: (v: string) => v ? (
      <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: '#434343' }}>{v}</code>
    ) : <span style={{ color: '#d9d9d9' }}>—</span>,
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
    width: 90,
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
