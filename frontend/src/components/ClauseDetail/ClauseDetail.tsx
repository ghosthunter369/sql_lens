import { Table, Tag, Empty } from 'antd'
import {
  SortAscendingOutlined,
  GroupOutlined,
  ColumnWidthOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import type { GroupByMeta, OrderByMeta } from '@/types/sql'

export default function ClauseDetail() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  const { groupBy, orderBy, limit } = result
  const hasContent = groupBy.length > 0 || orderBy.length > 0 || !!limit

  if (!hasContent) {
    return (
      <div className="tab-empty-state">
        <Empty description="当前 SQL 没有 GROUP BY / ORDER BY / LIMIT 子句" />
      </div>
    )
  }

  return (
    <div className="clause-detail-wrapper">
      {groupBy.length > 0 && (
        <div className="clause-section">
          <div className="clause-section-title">
            <GroupOutlined />
            <span>GROUP BY</span>
            <Tag color="blue">{groupBy.length} 个分组字段</Tag>
          </div>
          <Table<GroupByMeta>
            dataSource={groupBy}
            rowKey={(_, i) => `gb-${i}`}
            size="small"
            pagination={false}
            columns={[
              {
                title: '表达式',
                dataIndex: 'expression',
                key: 'expression',
                render: (text: string) => <code className="clause-expr">{text}</code>,
              },
              {
                title: '来源表',
                dataIndex: 'sourceTable',
                key: 'sourceTable',
                width: 120,
                render: (text?: string) => text || '-',
              },
            ]}
          />
        </div>
      )}

      {orderBy.length > 0 && (
        <div className="clause-section">
          <div className="clause-section-title">
            <SortAscendingOutlined />
            <span>ORDER BY</span>
            <Tag color="green">{orderBy.length} 个排序字段</Tag>
          </div>
          <Table<OrderByMeta>
            dataSource={orderBy}
            rowKey={(_, i) => `ob-${i}`}
            size="small"
            pagination={false}
            columns={[
              {
                title: '表达式',
                dataIndex: 'expression',
                key: 'expression',
                render: (text: string) => <code className="clause-expr">{text}</code>,
              },
              {
                title: '方向',
                dataIndex: 'direction',
                key: 'direction',
                width: 80,
                render: (dir: string) => (
                  <Tag color={dir === 'DESC' ? 'red' : 'blue'}>{dir}</Tag>
                ),
              },
              {
                title: '来源表',
                dataIndex: 'sourceTable',
                key: 'sourceTable',
                width: 120,
                render: (text?: string) => text || '-',
              },
            ]}
          />
        </div>
      )}

      {limit && (
        <div className="clause-section">
          <div className="clause-section-title">
            <ColumnWidthOutlined />
            <span>LIMIT</span>
          </div>
          <div className="clause-limit-info">
            <div className="clause-limit-item">
              <span className="clause-limit-label">LIMIT</span>
              <span className="clause-limit-value">{limit.limit}</span>
            </div>
            {limit.offset !== undefined && limit.offset > 0 && (
              <div className="clause-limit-item">
                <span className="clause-limit-label">OFFSET</span>
                <span className="clause-limit-value">{limit.offset}</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
