import type { TableMeta, JoinMeta, GraphNode, GraphEdge } from '@/types/sql'

export function buildGraphData(tables: TableMeta[], joins: JoinMeta[]): { nodes: GraphNode[]; edges: GraphEdge[] } {
  const tableNameToID: Record<string, string> = {}

  const nodes: GraphNode[] = tables.map((table, index) => {
    tableNameToID[table.name] = table.id
    return {
      id: table.id,
      type: 'tableNode',
      position: {
        x: index * 360,
        y: (index % 2) * 200,
      },
      data: {
        tableName: table.name,
        alias: table.alias,
        role: table.role,
        selectedFields: table.selectedFields,
        filterFields: table.filterFields,
        joinFields: table.joinFields,
      },
    }
  })

  const edges: GraphEdge[] = joins.map((join) => ({
    id: join.id,
    source: tableNameToID[join.leftTable] || join.leftTable,
    target: tableNameToID[join.rightTable] || join.rightTable,
    label: join.type,
    type: 'joinEdge',
    data: {
      conditions: join.conditions,
      rawExpr: join.rawExpr,
    },
  }))

  return { nodes, edges }
}
