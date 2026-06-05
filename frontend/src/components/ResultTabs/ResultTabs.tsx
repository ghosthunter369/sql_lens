import { Tabs, Badge } from 'antd'
import {
  ApartmentOutlined,
  TableOutlined,
  FilterOutlined,
  LinkOutlined,
  AlertOutlined,
  SortAscendingOutlined,
  NodeIndexOutlined,
  MergeCellsOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import ErrorBoundary from '@/components/ErrorBoundary'
import RelationGraph from '@/components/RelationGraph/RelationGraph'
import FieldTable from '@/components/FieldTable/FieldTable'
import WhereTree from '@/components/WhereTree/WhereTree'
import JoinDetail from '@/components/JoinDetail/JoinDetail'
import RiskPanel from '@/components/RiskPanel/RiskPanel'
import ClauseDetail from '@/components/ClauseDetail/ClauseDetail'

export default function ResultTabs() {
  const result = useSqlStore((s) => s.result)

  if (!result) return null

  const tabItems = [
    {
      key: 'graph',
      label: (
        <span className="tab-label">
          <ApartmentOutlined />
          <span>表关系图</span>
        </span>
      ),
      children: <ErrorBoundary><RelationGraph /></ErrorBoundary>,
    },
    {
      key: 'fields',
      label: (
        <span className="tab-label">
          <TableOutlined />
          <span>查询字段</span>
          <Badge count={result.fields.length} size="small" style={{ backgroundColor: '#1677ff' }} overflowCount={999} />
        </span>
      ),
      children: <ErrorBoundary><FieldTable /></ErrorBoundary>,
    },
    {
      key: 'where',
      label: (
        <span className="tab-label">
          <FilterOutlined />
          <span>WHERE 条件</span>
        </span>
      ),
      children: <ErrorBoundary><WhereTree /></ErrorBoundary>,
    },
    {
      key: 'joins',
      label: (
        <span className="tab-label">
          <LinkOutlined />
          <span>JOIN 明细</span>
          <Badge count={result.joins.length} size="small" style={{ backgroundColor: '#52c41a' }} overflowCount={99} />
        </span>
      ),
      children: <ErrorBoundary><JoinDetail /></ErrorBoundary>,
    },
    {
      key: 'clauses',
      label: (
        <span className="tab-label">
          <SortAscendingOutlined />
          <span>排序与分组</span>
        </span>
      ),
      children: <ErrorBoundary><ClauseDetail /></ErrorBoundary>,
    },
    {
      key: 'risks',
      label: (
        <span className="tab-label">
          <AlertOutlined />
          <span>风险提示</span>
          {result.risks.length > 0 && (
            <Badge count={result.risks.length} size="small" style={{ backgroundColor: '#ff4d4f' }} overflowCount={99} />
          )}
        </span>
      ),
      children: <ErrorBoundary><RiskPanel /></ErrorBoundary>,
    },
    ...(result.ctes && result.ctes.length > 0 ? [{
      key: 'ctes',
      label: (
        <span className="tab-label">
          <NodeIndexOutlined />
          <span>CTE</span>
          <Badge count={result.ctes.length} size="small" style={{ backgroundColor: '#722ed1' }} />
        </span>
      ),
      children: (
        <ErrorBoundary>
          <div style={{ padding: 12 }}>
            {result.ctes.map((cte) => (
              <div key={cte.id} style={{ marginBottom: 16 }}>
                <div style={{ fontWeight: 600, marginBottom: 4, color: '#722ed1' }}>
                  {cte.name}
                  {cte.columns && cte.columns.length > 0 && (
                    <span style={{ fontWeight: 400, color: '#666', marginLeft: 8 }}>
                      ({cte.columns.join(', ')})
                    </span>
                  )}
                </div>
                <pre style={{
                  background: '#f5f5f5',
                  padding: 8,
                  borderRadius: 6,
                  fontSize: 12,
                  fontFamily: 'var(--font-mono)',
                  overflow: 'auto',
                  maxHeight: 200,
                }}>
                  {cte.rawSql}
                </pre>
              </div>
            ))}
          </div>
        </ErrorBoundary>
      ),
    }] : []),
    ...(result.setOperations && result.setOperations.length > 0 ? [{
      key: 'set-ops',
      label: (
        <span className="tab-label">
          <MergeCellsOutlined />
          <span>集合操作</span>
          <Badge count={result.setOperations.length} size="small" style={{ backgroundColor: '#13c2c2' }} />
        </span>
      ),
      children: (
        <ErrorBoundary>
          <div style={{ padding: 12 }}>
            {result.setOperations.map((op, idx) => (
              <div key={idx} style={{ marginBottom: 16 }}>
                <div style={{ fontWeight: 600, marginBottom: 8, color: '#13c2c2', fontSize: 14 }}>
                  {op.type}
                </div>
                <div style={{ display: 'flex', gap: 12 }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>左侧查询</div>
                    <pre style={{
                      background: '#f5f5f5',
                      padding: 8,
                      borderRadius: 6,
                      fontSize: 12,
                      fontFamily: 'var(--font-mono)',
                      overflow: 'auto',
                      maxHeight: 150,
                    }}>
                      {op.left?.rawSql || '(无数据)'}
                    </pre>
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>右侧查询</div>
                    <pre style={{
                      background: '#f5f5f5',
                      padding: 8,
                      borderRadius: 6,
                      fontSize: 12,
                      fontFamily: 'var(--font-mono)',
                      overflow: 'auto',
                      maxHeight: 150,
                    }}>
                      {op.right?.rawSql || '(无数据)'}
                    </pre>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </ErrorBoundary>
      ),
    }] : []),
  ]

  return (
    <div style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
      <Tabs
        defaultActiveKey="graph"
        items={tabItems}
        size="small"
        style={{ height: '100%' }}
        tabBarStyle={{ marginBottom: 0, padding: '0 8px' }}
      />
    </div>
  )
}
