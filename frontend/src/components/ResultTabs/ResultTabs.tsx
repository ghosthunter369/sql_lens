import { Tabs, Badge } from 'antd'
import {
  ApartmentOutlined,
  TableOutlined,
  FilterOutlined,
  LinkOutlined,
  AlertOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import RelationGraph from '@/components/RelationGraph/RelationGraph'
import FieldTable from '@/components/FieldTable/FieldTable'
import WhereTree from '@/components/WhereTree/WhereTree'
import JoinDetail from '@/components/JoinDetail/JoinDetail'
import RiskPanel from '@/components/RiskPanel/RiskPanel'

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
      children: <RelationGraph />,
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
      children: <FieldTable />,
    },
    {
      key: 'where',
      label: (
        <span className="tab-label">
          <FilterOutlined />
          <span>WHERE 条件</span>
        </span>
      ),
      children: <WhereTree />,
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
      children: <JoinDetail />,
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
      children: <RiskPanel />,
    },
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
