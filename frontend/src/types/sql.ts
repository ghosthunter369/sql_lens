export type Dialect = 'mysql' | 'postgresql' | 'oracle' | 'sqlserver'
export type LogType = 'auto' | 'plain' | 'laravel' | 'mybatis' | 'thinkphp'

export interface ParseSQLRequest {
  rawText: string
  dialect: Dialect
  logType: LogType
  options: {
    restoreBindings: boolean
    formatSql: boolean
    enableRiskCheck: boolean
  }
}

export interface SQLAnalysisResult {
  statementType: string
  dialect: string
  rawSql: string
  restoredSql: string
  formattedSql: string
  summary: SQLSummary
  tables: TableMeta[]
  joins: JoinMeta[]
  fields: FieldMeta[]
  whereTree?: ConditionNode
  groupBy: GroupByMeta[]
  orderBy: OrderByMeta[]
  limit?: LimitMeta
  graph: GraphMeta
  risks: RiskMeta[]
}

export interface SQLSummary {
  tableCount: number
  joinCount: number
  fieldCount: number
  whereCount: number
  hasGroupBy: boolean
  hasOrderBy: boolean
  hasLimit: boolean
  complexity: 'LOW' | 'MEDIUM' | 'HIGH'
}

export interface TableMeta {
  id: string
  name: string
  alias: string
  role: 'main' | 'joined' | 'subquery' | 'cte' | 'derived'
  selectedFields: string[]
  filterFields: string[]
  joinFields: string[]
}

export interface JoinMeta {
  id: string
  type: string
  leftTable: string
  rightTable: string
  conditions: JoinCondition[]
  rawExpr: string
}

export interface JoinCondition {
  left: string
  operator: string
  right: string
}

export interface FieldMeta {
  id: string
  outputName: string
  sourceTable: string
  sourceAlias: string
  sourceColumn: string
  expression: string
  fieldType: 'column' | 'function' | 'aggregate' | 'case' | 'subquery' | 'wildcard'
}

export interface ConditionNode {
  id: string
  type: 'AND' | 'OR' | 'CONDITION'
  expr?: string
  table?: string
  field?: string
  operator?: string
  value?: string
  children?: ConditionNode[]
}

export interface GraphMeta {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export interface GraphNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: Record<string, unknown>
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  label: string
  type: string
  data: Record<string, unknown>
}

export interface RiskMeta {
  id: string
  level: 'info' | 'warning' | 'danger'
  type: string
  message: string
  suggestion: string
  relatedExpr?: string
}

export interface GroupByMeta {
  expression: string
  sourceTable?: string
}

export interface OrderByMeta {
  expression: string
  sourceTable?: string
  direction: 'ASC' | 'DESC'
}

export interface LimitMeta {
  limit: number
  offset?: number
}

export interface APIResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
    detail?: string
  }
}
