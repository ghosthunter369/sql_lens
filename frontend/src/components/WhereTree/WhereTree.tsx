import { Tree, Tag } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { useSqlStore } from '@/store/useSqlStore'
import type { ConditionNode } from '@/types/sql'

function buildTreeNodes(node: ConditionNode): DataNode {
  if (node.type === 'CONDITION') {
    return {
      key: node.id,
      title: (
        <span className="where-condition-item">
          <code className="where-condition-expr">{node.expr}</code>
          {node.table && (
            <Tag style={{ marginLeft: 8, fontSize: 10, fontFamily: 'var(--font-mono)' }}>
              {node.table}
            </Tag>
          )}
        </span>
      ),
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
    children: (node.children || []).map(buildTreeNodes),
    selectable: false,
  }
}

export default function WhereTree() {
  const result = useSqlStore((s) => s.result)
  if (!result) return null

  if (!result.whereTree) {
    return (
      <div className="tab-empty-state">
        <span>暂无 WHERE 条件</span>
      </div>
    )
  }

  const treeData = [buildTreeNodes(result.whereTree)]

  const countConditions = (node: ConditionNode): number => {
    if (node.type === 'CONDITION') return 1
    return (node.children || []).reduce((sum, c) => sum + countConditions(c), 0)
  }

  const totalConditions = countConditions(result.whereTree)

  return (
    <div className="where-tree-wrapper">
      <div className="tab-section-header">
        共 <strong>{totalConditions}</strong> 个条件
      </div>
      <Tree
        treeData={treeData}
        defaultExpandAll
        showLine={{ showLeafIcon: false }}
        style={{ fontSize: 13 }}
      />
    </div>
  )
}
