import { SearchOutlined } from '@ant-design/icons'

export default function EmptyState() {
  return (
    <div className="sql-lens-empty">
      <div className="sql-lens-empty-icon">
        <SearchOutlined />
      </div>
      <h2>SQL Lens — SQL 透视镜</h2>
      <p>
        将复杂的 SQL 或日志粘贴到左侧编辑器，
        <br />
        一键解析出表关系图、字段来源、WHERE 条件树和风险提示。
      </p>
      <p style={{ marginTop: 12, fontSize: 12 }}>
        按 <kbd>Ctrl+Enter</kbd> 快速解析
      </p>
    </div>
  )
}
