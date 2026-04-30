import { Tag } from 'antd'
import { useSqlStore } from '@/store/useSqlStore'

const joinTypeColors: Record<string, string> = {
  'LEFT JOIN': '#1677ff',
  'RIGHT JOIN': '#722ed1',
  'INNER JOIN': '#52c41a',
  'JOIN': '#52c41a',
  'CROSS JOIN': '#fa8c16',
  'LEFT OUTER JOIN': '#1677ff',
  'RIGHT OUTER JOIN': '#722ed1',
}

export default function JoinDetail() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  if (result.joins.length === 0) {
    return (
      <div className="tab-empty-state">
        <span>暂无 JOIN 关系</span>
      </div>
    )
  }

  return (
    <div className="join-detail-wrapper">
      <div className="tab-section-header">
        共 <strong>{result.joins.length}</strong> 个 JOIN
      </div>
      {result.joins.map((join) => (
        <div key={join.id} className="join-card">
          <div className="join-card-header">
            <Tag color={joinTypeColors[join.type] || 'default'} style={{ fontWeight: 600 }}>
              {join.type}
            </Tag>
            <span className="join-tables">
              <strong>{join.leftTable}</strong>
              <span className="join-arrow">→</span>
              <strong>{join.rightTable}</strong>
            </span>
          </div>
          <div className="join-card-body">
            <div className="join-meta">
              <div className="join-meta-item">
                <span className="join-meta-label">左表</span>
                <span className="join-meta-value">{join.leftTable}</span>
              </div>
              <div className="join-meta-item">
                <span className="join-meta-label">右表</span>
                <span className="join-meta-value">{join.rightTable}</span>
              </div>
              <div className="join-meta-item">
                <span className="join-meta-label">JOIN 类型</span>
                <Tag style={{ margin: 0, fontWeight: 500 }} color={joinTypeColors[join.type] || 'default'}>
                  {join.type}
                </Tag>
              </div>
              <div className="join-meta-item">
                <span className="join-meta-label">条件数</span>
                <span className="join-meta-value">{join.conditions.length}</span>
              </div>
            </div>
            {join.conditions.length > 0 && (
              <div className="join-conditions">
                <div className="join-conditions-label">ON 条件</div>
                {join.conditions.map((cond, ci) => (
                  <code key={ci} className="join-on-code">
                    {cond.left} <span className="join-op">{cond.operator}</span> {cond.right}
                  </code>
                ))}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
