import {
  TableOutlined,
  LinkOutlined,
  FieldStringOutlined,
  FilterOutlined,
  ApartmentOutlined,
  ThunderboltOutlined,
  NodeIndexOutlined,
  FunctionOutlined,
  MergeCellsOutlined,
  FileTextOutlined,
} from '@ant-design/icons'
import { Tag } from 'antd'
import { useSqlStore } from '@/store/useSqlStore'

const complexityInfo: Record<string, { icon: React.ReactNode; label: string; className: string }> = {
  LOW: { icon: <ThunderboltOutlined />, label: '低', className: 'green' },
  MEDIUM: { icon: <ThunderboltOutlined />, label: '中', className: 'orange' },
  HIGH: { icon: <ThunderboltOutlined />, label: '高', className: 'red' },
}

export default function SummaryCards() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  const { summary } = result
  const comp = complexityInfo[summary.complexity] || complexityInfo.LOW

  const items = [
    { icon: <TableOutlined />, value: summary.tableCount, label: '涉及表', unit: '张', className: 'blue' },
    { icon: <LinkOutlined />, value: summary.joinCount, label: 'JOIN 关系', unit: '个', className: 'cyan' },
    { icon: <FieldStringOutlined />, value: summary.fieldCount, label: '查询字段', unit: '个', className: 'green' },
    { icon: <FilterOutlined />, value: summary.whereCount, label: 'WHERE 条件', unit: '个', className: 'orange' },
    { icon: <ApartmentOutlined />, value: result.statementType, label: 'SQL 类型', unit: '', className: 'purple', isString: true },
    {
      icon: comp.icon,
      value: comp.label,
      label: '复杂度',
      unit: '',
      className: comp.className,
      isString: true,
    },
  ]

  // Feature tags
  const featureTags = []
  if (summary.hasCTE) {
    featureTags.push({ label: 'CTE', color: '#722ed1', icon: <NodeIndexOutlined /> })
  }
  if (summary.hasWindowFunc) {
    featureTags.push({ label: '窗口函数', color: '#1677ff', icon: <FunctionOutlined /> })
  }
  if (summary.hasUnion) {
    featureTags.push({ label: 'UNION', color: '#13c2c2', icon: <MergeCellsOutlined /> })
  }
  if (summary.hasHaving) {
    featureTags.push({ label: 'HAVING', color: '#fa8c16', icon: <FileTextOutlined /> })
  }

  return (
    <div className="sql-lens-summary">
      <div className="sql-lens-summary-grid">
        {items.map((item, i) => (
          <div key={i} className="sql-lens-summary-card">
            <div className={`sql-lens-summary-card-icon ${item.className}`}>{item.icon}</div>
            <div className="sql-lens-summary-card-info">
              <span className="sql-lens-summary-card-value">
                {item.value}
                {item.unit && (
                  <span className="sql-lens-summary-card-unit">
                    {item.unit}
                  </span>
                )}
              </span>
              <span className="sql-lens-summary-card-label">{item.label}</span>
            </div>
          </div>
        ))}
      </div>
      {featureTags.length > 0 && (
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 8 }}>
          {featureTags.map((tag, i) => (
            <Tag key={i} color={tag.color} icon={tag.icon} style={{ borderRadius: 4, fontSize: 11 }}>
              {tag.label}
            </Tag>
          ))}
        </div>
      )}
    </div>
  )
}
