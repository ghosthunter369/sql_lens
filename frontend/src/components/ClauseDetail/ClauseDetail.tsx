import { Table, Tag, Empty, Tree } from 'antd'
import type { DataNode } from 'antd/es/tree'
import {
  SortAscendingOutlined,
  GroupOutlined,
  ColumnWidthOutlined,
  FilterOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'
import type { GroupByMeta, OrderByMeta, ConditionNode } from '@/types/sql'

function buildHavingTreeNodes(node: ConditionNode): DataNode {
  if (node.type === 'CONDITION') {
    return {
      key: node.id,
      title: <code style={{ fontSize: 12 }}>{node.expr}</code>,
      isLeaf: true,
      selectable: false,
    }
  }
  const isAnd = node.type === 'AND'
  return {
    key: node.id,
    title: (
      <span className={`where-logic-tag ${isAnd ? 'and' : 'or'}`}>
        {node.type}
      </span>
    ),
    children: (node.children || []).map(buildHavingTreeNodes),
    selectable: false,
  }
}

export default function ClauseDetail() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  const { groupBy, orderBy, limit, havingTree } = result
  const hasContent = groupBy.length > 0 || orderBy.length > 0 || !!limit || !!havingTree

  if (!hasContent) {
    return (
      <div className="tab-empty-state">
        <Empty description="当前 SQL 没有 GROUP BY / ORDER BY / LIMIT / HAVING 子句" />
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

      {havingTree && (
        <div className="clause-section">
          <div className="clause-section-title">
            <FilterOutlined />
            <span>HAVING</span>
          </div>
          <Tree
            treeData={[buildHavingTreeNodes(havingTree)]}
            defaultExpandAll
            showLine={{ showLeafIcon: false }}
            style={{ fontSize: 13 }}
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
                  <Tag color={dir.toUpperCase() === 'DESC' ? 'red' : 'blue'}>{dir}</Tag>
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
