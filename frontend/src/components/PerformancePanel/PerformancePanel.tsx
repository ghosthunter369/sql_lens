import { Tag, Empty } from 'antd'
import {
  ThunderboltOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import type { SQLAnalysisResult, ConditionNode } from '@/types/sql'

interface Suggestion {
  type: 'index' | 'query' | 'join' | 'where' | 'general'
  level: 'critical' | 'warning' | 'info'
  title: string
  detail: string
  sql?: string
}

function analyzePerformance(result: SQLAnalysisResult): Suggestion[] {
  const suggestions: Suggestion[] = []

  // 1. Check for SELECT *
  const hasWildcard = result.fields.some((f) => f.fieldType === 'wildcard')
  if (hasWildcard) {
    suggestions.push({
      type: 'query',
      level: 'warning',
      title: '避免使用 SELECT *',
      detail: '明确指定需要的字段可以减少数据传输量，提高查询效率，特别是在大表上。',
    })
  }

  // 2. Check for missing WHERE on UPDATE/DELETE
  if ((result.statementType === 'UPDATE' || result.statementType === 'DELETE') && !result.whereTree) {
    suggestions.push({
      type: 'query',
      level: 'critical',
      title: `${result.statementType} 缺少 WHERE 条件`,
      detail: '没有 WHERE 条件的 UPDATE/DELETE 会影响全表数据，这是非常危险的操作。',
    })
  }

  // 3. Check JOIN count
  if (result.summary.joinCount >= 4) {
    suggestions.push({
      type: 'join',
      level: 'warning',
      title: `JOIN 数量过多 (${result.summary.joinCount})`,
      detail: '过多的 JOIN 会显著降低查询性能。考虑：1) 是否可以拆分为多个简单查询 2) 是否有冗余的 JOIN 3) 是否可以使用子查询替代。',
    })
  }

  // 4. Check for functions on WHERE columns
  if (result.whereTree) {
    checkWhereFunctions(result.whereTree, suggestions)
  }

  // 5. Check for LIKE with prefix wildcard
  if (result.whereTree) {
    checkLikeWildcard(result.whereTree, suggestions)
  }

  // 6. Check for subquery depth
  const subqueryCount = (result.rawSql.match(/\(\s*SELECT/gi) || []).length
  if (subqueryCount >= 2) {
    suggestions.push({
      type: 'query',
      level: 'warning',
      title: `子查询嵌套较深 (${subqueryCount} 层)`,
      detail: '深层子查询会影响可读性和性能。考虑使用 CTE (WITH 子句) 改写，使逻辑更清晰。',
    })
  }

  // 7. Check for ORDER BY RAND()
  const hasRand = result.orderBy.some((o) => o.expression.toUpperCase().includes('RAND()'))
  if (hasRand) {
    suggestions.push({
      type: 'query',
      level: 'critical',
      title: '避免 ORDER BY RAND()',
      detail: 'ORDER BY RAND() 在大表上性能极差，需要全表扫描和排序。建议在应用层实现随机选取。',
    })
  }

  // 8. Check for OR conditions
  if (result.whereTree) {
    const orCount = countORNodes(result.whereTree)
    if (orCount >= 3) {
      suggestions.push({
        type: 'where',
        level: 'info',
        title: `多个 OR 条件 (${orCount} 个)`,
        detail: '大量 OR 条件可能导致索引失效。考虑改写为 UNION 或 IN 子句。',
      })
    }
  }

  // 9. Index suggestions based on WHERE fields
  const whereFields = collectWhereFields(result.whereTree)
  if (whereFields.length > 0) {
    const fieldList = [...new Set(whereFields)].join(', ')
    suggestions.push({
      type: 'index',
      level: 'info',
      title: '索引建议',
      detail: `WHERE 条件涉及字段: ${fieldList}。确保这些字段上有合适的索引。对于复合条件，考虑创建复合索引。`,
    })
  }

  // 10. Check for DISTINCT
  const hasDistinct = result.fields.some((f) => f.expression.toUpperCase().includes('DISTINCT'))
  if (hasDistinct) {
    suggestions.push({
      type: 'query',
      level: 'info',
      title: '使用了 DISTINCT',
      detail: 'DISTINCT 通常意味着可能有重复数据。检查是否可以通过优化 JOIN 条件来避免重复，而不是依赖 DISTINCT 去重。',
    })
  }

  // 11. GROUP BY without LIMIT
  if (result.summary.hasGroupBy && !result.summary.hasLimit) {
    suggestions.push({
      type: 'query',
      level: 'info',
      title: 'GROUP BY 缺少 LIMIT',
      detail: '如果分组结果集可能很大，建议添加 LIMIT 限制返回行数。',
    })
  }

  // 12. Window function suggestions
  if (result.summary.hasWindowFunc) {
    suggestions.push({
      type: 'general',
      level: 'info',
      title: '窗口函数优化',
      detail: '窗口函数在大数据集上可能较慢。确保 PARTITION BY 和 ORDER BY 的字段有索引。',
    })
  }

  return suggestions
}

function checkWhereFunctions(node: ConditionNode, suggestions: Suggestion[]) {
  if (node.type === 'CONDITION') {
    if (node.expr && /\b\w+\s*\(/.test(node.expr)) {
      suggestions.push({
        type: 'where',
        level: 'warning',
        title: 'WHERE 中使用了函数',
        detail: `条件 "${node.expr}" 在字段上使用了函数，这会导致索引失效。考虑将函数移到值的一侧，或使用函数索引。`,
      })
    }
  }
  for (const child of node.children || []) {
    checkWhereFunctions(child, suggestions)
  }
}

function checkLikeWildcard(node: ConditionNode, suggestions: Suggestion[]) {
  if (node.type === 'CONDITION' && (node.operator === 'LIKE' || node.operator === 'NOT LIKE')) {
    const val = (node.value || '').replace(/['"]/g, '')
    if (val.startsWith('%') || val.startsWith('_')) {
      suggestions.push({
        type: 'index',
        level: 'warning',
        title: 'LIKE 前缀通配符',
        detail: `条件 "${node.expr}" 使用了前缀通配符 (${val.slice(0, 10)}...)，这会导致索引完全失效。`,
      })
    }
  }
  for (const child of node.children || []) {
    checkLikeWildcard(child, suggestions)
  }
}

function countORNodes(node: ConditionNode): number {
  if (!node) return 0
  let count = node.type === 'OR' ? 1 : 0
  for (const child of node.children || []) {
    count += countORNodes(child)
  }
  return count
}

function collectWhereFields(node: ConditionNode | undefined): string[] {
  if (!node) return []
  const fields: string[] = []
  if (node.type === 'CONDITION' && node.field) {
    fields.push(node.field)
  }
  for (const child of node.children || []) {
    fields.push(...collectWhereFields(child))
  }
  return fields
}

const levelConfig = {
  critical: { color: '#ff4d4f', bg: '#fff2f0', icon: <WarningOutlined />, label: '严重' },
  warning: { color: '#fa8c16', bg: '#fff7e6', icon: <WarningOutlined />, label: '警告' },
  info: { color: '#1677ff', bg: '#e6f4ff', icon: <InfoCircleOutlined />, label: '建议' },
}

const typeLabels: Record<string, string> = {
  index: '索引',
  query: '查询',
  join: 'JOIN',
  where: 'WHERE',
  general: '通用',
}

const typeColors: Record<string, string> = {
  index: 'purple',
  query: 'blue',
  join: 'cyan',
  where: 'orange',
  general: 'default',
}

export default function PerformancePanel() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  const suggestions = analyzePerformance(result)

  if (suggestions.length === 0) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}>
        <CheckCircleOutlined style={{ fontSize: 40, color: '#52c41a', marginBottom: 12 }} />
        <div style={{ color: '#52c41a', fontWeight: 600 }}>SQL 质量良好</div>
        <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>未发现明显的性能问题</div>
      </div>
    )
  }

  const critical = suggestions.filter((s) => s.level === 'critical')
  const warnings = suggestions.filter((s) => s.level === 'warning')
  const infos = suggestions.filter((s) => s.level === 'info')

  return (
    <div style={{ padding: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <ThunderboltOutlined style={{ color: '#fa8c16' }} />
        <span style={{ fontWeight: 600, fontSize: 14 }}>性能建议</span>
        <Tag color="red">{critical.length} 严重</Tag>
        <Tag color="orange">{warnings.length} 警告</Tag>
        <Tag color="blue">{infos.length} 建议</Tag>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {[...critical, ...warnings, ...infos].map((s, i) => {
          const cfg = levelConfig[s.level]
          return (
            <div
              key={i}
              style={{
                padding: '10px 12px',
                borderRadius: 8,
                border: `1px solid ${cfg.color}20`,
                background: cfg.bg,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                <span style={{ color: cfg.color }}>{cfg.icon}</span>
                <span style={{ fontWeight: 600, fontSize: 13, color: cfg.color }}>{s.title}</span>
                <Tag color={typeColors[s.type]} style={{ fontSize: 10 }}>{typeLabels[s.type]}</Tag>
              </div>
              <div style={{ fontSize: 12, color: '#666', lineHeight: 1.6 }}>{s.detail}</div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
