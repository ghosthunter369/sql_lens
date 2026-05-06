import { Alert, Tag } from 'antd'
import {
  WarningOutlined,
  InfoCircleOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import type { RiskMeta } from '@/types/sql'

const levelConfig: Record<string, { icon: React.ReactNode; borderColor: string; tagColor: string }> = {
  info:    { icon: <InfoCircleOutlined />, borderColor: '#1677ff', tagColor: 'blue' },
  warning: { icon: <WarningOutlined />,      borderColor: '#faad14', tagColor: 'orange' },
  danger:  { icon: <StopOutlined />,          borderColor: '#ff4d4f', tagColor: 'red' },
}

const riskTypeLabels: Record<string, string> = {
  SELECT_STAR: 'SELECT *',
  NO_WHERE_UPDATE: 'UPDATE 无 WHERE',
  NO_WHERE_DELETE: 'DELETE 无 WHERE',
  LIKE_PREFIX_WILDCARD: 'LIKE 前缀通配',
  WHERE_FIELD_FUNCTION: 'WHERE 函数',
  TOO_MANY_JOINS: 'JOIN 过多',
  ORDER_BY_RAND: 'ORDER BY RAND',
  TOO_MANY_OR: 'OR 过多',
  SUBQUERY_DEPTH: '子查询嵌套深',
}

export default function RiskPanel() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  if (result.risks.length === 0) {
    return (
      <Alert
        message="未发现明显风险"
        description="当前 SQL 通过基础风险检查"
        type="success"
        showIcon
        style={{ borderRadius: 8 }}
      />
    )
  }

  return (
    <div className="risk-panel-wrapper">
      <div className="tab-section-header">
        共 <strong>{result.risks.length}</strong> 条风险提示
      </div>
      {result.risks.map((risk: RiskMeta) => {
        const config = levelConfig[risk.level] || levelConfig.info
        return (
          <div key={risk.id} className={`risk-card ${risk.level}`} style={{ borderLeftColor: config.borderColor }}>
            <div className="risk-card-header">
              <span className="risk-card-icon">{config.icon}</span>
              <Tag color={config.tagColor} style={{ fontWeight: 600 }}>{riskTypeLabels[risk.type] || risk.type}</Tag>
              <span className="risk-card-message">{risk.message}</span>
            </div>
            <div className="risk-card-body">
              <div className="risk-card-suggestion">{risk.suggestion}</div>
              {risk.relatedExpr && (
                <code className="risk-card-expr">{risk.relatedExpr}</code>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
