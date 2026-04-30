import {
  TableOutlined,
  LinkOutlined,
  FieldStringOutlined,
  FilterOutlined,
  ApartmentOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
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

  return (
    <div className="sql-lens-summary">
      <div className="sql-lens-summary-grid">
        {items.map((item, i) => (
          <div key={i} className="sql-lens-summary-card">
            <div className={`sql-lens-summary-card-icon ${item.className}`}>{item.icon}</div>
            <div className="sql-lens-summary-card-info">
              <span className="sql-lens-summary-card-value">
                {item.isString ? item.value : item.value}
                {item.unit && (
                  <span style={{ fontSize: 12, fontWeight: 400, color: '#999', marginLeft: 2 }}>
                    {item.unit}
                  </span>
                )}
              </span>
              <span className="sql-lens-summary-card-label">{item.label}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
