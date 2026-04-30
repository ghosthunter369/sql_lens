import { memo } from 'react'
import { BaseEdge, EdgeLabelRenderer, getBezierPath, type EdgeProps } from '@xyflow/react'

interface JoinCondition {
  left: string
  operator: string
  right: string
}

interface JoinEdgeData {
  conditions?: JoinCondition[]
  rawExpr?: string
}

function formatCondition(c: JoinCondition): string {
  return `${c.left} ${c.operator} ${c.right}`
}

function JoinEdge({
  id, sourceX, sourceY, targetX, targetY,
  sourcePosition, targetPosition, data, label,
}: EdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX, sourceY, sourcePosition,
    targetX, targetY, targetPosition,
  })

  const edgeData = data as JoinEdgeData | undefined
  const labelStr = typeof label === 'string' ? label : ''

  // Build condition label — show up to 2 conditions compactly
  let condLabel = ''
  if (edgeData?.conditions && edgeData.conditions.length > 0) {
    const conds = edgeData.conditions
    condLabel = conds.slice(0, 2).map(formatCondition).join(', ')
    if (conds.length > 2) condLabel += ` (+${conds.length - 2})`
  } else if (edgeData?.rawExpr) {
    condLabel = edgeData.rawExpr
  }

  // Truncate if too long for edge display, full text in title
  const displayCond = condLabel.length > 32 ? condLabel.substring(0, 30) + '…' : condLabel

  const joinColor = labelStr.includes('LEFT') ? '#1677ff'
    : labelStr.includes('RIGHT') ? '#722ed1'
    : labelStr.includes('CROSS') ? '#fa8c16'
    : '#52c41a'

  const isOuter = labelStr.includes('OUTER')

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: joinColor,
          strokeWidth: 2.5,
          strokeDasharray: isOuter ? '6 3' : 'none',
        }}
      />
      {condLabel && (
        <EdgeLabelRenderer>
          <div
            className="join-edge-label"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              background: joinColor + '15',
              borderColor: joinColor + '40',
              color: joinColor,
            }}
            title={condLabel}
          >
            {displayCond}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

export default memo(JoinEdge)
