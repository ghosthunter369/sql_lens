import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { FieldStringOutlined, FilterOutlined, LinkOutlined } from '@ant-design/icons'

interface TableNodeData {
  tableName: string
  alias: string
  role: string
  selectedFields: string[]
  filterFields: string[]
  joinFields: string[]
}

const roleLabels: Record<string, string> = {
  main: '主表',
  joined: 'JOIN表',
  subquery: '子查询',
  cte: 'CTE',
  derived: '派生表',
}

function FieldTags({ fields, icon, max }: { fields: string[]; icon: React.ReactNode; max: number }) {
  if (!fields || fields.length === 0) return null
  const show = fields.slice(0, max)
  const more = fields.length - max
  return (
    <div className="table-node-field-row">
      <span className="table-node-field-icon">{icon}</span>
      <div className="table-node-field-tags">
        {show.map((f, i) => (
          <span key={i} className="table-node-field-tag">{f}</span>
        ))}
        {more > 0 && <span className="table-node-field-more">+{more}</span>}
      </div>
    </div>
  )
}

function TableNode({ data }: { data: TableNodeData }) {
  const sf = data.selectedFields?.length || 0
  const ff = data.filterFields?.length || 0
  const jf = data.joinFields?.length || 0

  return (
    <div className="table-node">
      <Handle type="target" position={Position.Top} style={{ background: '#1677ff', width: 8, height: 8 }} />
      <Handle type="source" position={Position.Bottom} style={{ background: '#1677ff', width: 8, height: 8 }} />

      <div className={`table-node-header ${data.role}`}>
        <span className="table-node-name">{data.tableName}</span>
        <span className="table-node-role">{roleLabels[data.role] || data.role}</span>
      </div>

      <div className="table-node-body">
        {data.alias && (
          <div className="table-node-alias">
            别名: <strong>{data.alias}</strong>
          </div>
        )}

        {sf === 0 && ff === 0 && jf === 0 ? (
          <div className="table-node-empty-meta">{data.role === 'subquery' ? '子查询' : data.role === 'cte' ? 'CTE 引用' : '暂无字段'}</div>
        ) : (
          <>
            <FieldTags fields={data.selectedFields} icon={<FieldStringOutlined />} max={4} />
            <FieldTags fields={data.joinFields} icon={<LinkOutlined />} max={3} />
            <FieldTags fields={data.filterFields} icon={<FilterOutlined />} max={3} />
          </>
        )}
      </div>
    </div>
  )
}

export default memo(TableNode)
