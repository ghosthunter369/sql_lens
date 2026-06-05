import { useState } from 'react'
import { List, Tag, Button, Input, Empty, Tooltip, Collapse } from 'antd'
import {
  CodeOutlined,
  PlusOutlined,
  DeleteOutlined,
  CopyOutlined,
  BookOutlined,
} from '@ant-design/icons'
import { useSqlStore } from '@/store/useSqlStore'

interface Snippet {
  id: string
  name: string
  sql: string
  category: string
  builtin: boolean
}

const STORAGE_KEY = 'sql-lens-snippets'

const builtinSnippets: Snippet[] = [
  {
    id: 'basic-select',
    name: '基础 SELECT',
    sql: 'SELECT id, name, email\nFROM users\nWHERE status = 1\nORDER BY created_at DESC\nLIMIT 10;',
    category: '基础查询',
    builtin: true,
  },
  {
    id: 'join-query',
    name: '多表 JOIN',
    sql: 'SELECT u.name, o.order_no, o.amount\nFROM users u\nINNER JOIN orders o ON u.id = o.user_id\nLEFT JOIN products p ON o.product_id = p.id\nWHERE o.status = \'paid\'\nORDER BY o.created_at DESC;',
    category: '基础查询',
    builtin: true,
  },
  {
    id: 'subquery',
    name: '子查询示例',
    sql: 'SELECT *\nFROM products\nWHERE category_id IN (\n  SELECT id FROM categories\n  WHERE parent_id = 10\n)\nAND price > (\n  SELECT AVG(price) FROM products\n);',
    category: '高级查询',
    builtin: true,
  },
  {
    id: 'cte-example',
    name: 'CTE 公用表表达式',
    sql: 'WITH monthly_sales AS (\n  SELECT\n    DATE_FORMAT(created_at, \'%Y-%m\') AS month,\n    SUM(amount) AS total\n  FROM orders\n  WHERE status = \'paid\'\n  GROUP BY month\n)\nSELECT\n  month,\n  total,\n  LAG(total) OVER (ORDER BY month) AS prev_month\nFROM monthly_sales\nORDER BY month;',
    category: '高级查询',
    builtin: true,
  },
  {
    id: 'window-function',
    name: '窗口函数',
    sql: 'SELECT\n  name,\n  department,\n  salary,\n  ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) AS rank,\n  AVG(salary) OVER (PARTITION BY department) AS dept_avg\nFROM employees;',
    category: '高级查询',
    builtin: true,
  },
  {
    id: 'union-example',
    name: 'UNION 合并查询',
    sql: 'SELECT name, email, \'active\' AS status\nFROM users WHERE status = 1\nUNION ALL\nSELECT name, email, \'inactive\' AS status\nFROM users WHERE status = 0;',
    category: '高级查询',
    builtin: true,
  },
  {
    id: 'insert-select',
    name: 'INSERT INTO SELECT',
    sql: 'INSERT INTO user_archive (id, name, email)\nSELECT id, name, email\nFROM users\nWHERE last_login < DATE_SUB(NOW(), INTERVAL 1 YEAR);',
    category: 'DML',
    builtin: true,
  },
  {
    id: 'update-join',
    name: 'UPDATE JOIN',
    sql: 'UPDATE orders o\nINNER JOIN users u ON o.user_id = u.id\nSET o.discount = 0.1\nWHERE u.vip_level >= 3;',
    category: 'DML',
    builtin: true,
  },
  {
    id: 'group-by-having',
    name: 'GROUP BY + HAVING',
    sql: 'SELECT\n  user_id,\n  COUNT(*) AS order_count,\n  SUM(amount) AS total_amount\nFROM orders\nWHERE status = \'paid\'\nGROUP BY user_id\nHAVING total_amount > 1000\nORDER BY total_amount DESC;',
    category: '聚合查询',
    builtin: true,
  },
  {
    id: 'exists-check',
    name: 'EXISTS 检查',
    sql: 'SELECT u.name\nFROM users u\nWHERE EXISTS (\n  SELECT 1 FROM orders o\n  WHERE o.user_id = u.id\n  AND o.created_at > DATE_SUB(NOW(), INTERVAL 30 DAY)\n);',
    category: '高级查询',
    builtin: true,
  },
]

function loadCustomSnippets(): Snippet[] {
  try {
    const data = localStorage.getItem(STORAGE_KEY)
    return data ? JSON.parse(data) : []
  } catch {
    return []
  }
}

function saveCustomSnippets(snippets: Snippet[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(snippets))
  } catch {
    // ignore
  }
}

export default function SnippetPanel() {
  const setRawText = useSqlStore((s) => s.setRawText)
  const [customSnippets, setCustomSnippets] = useState<Snippet[]>(loadCustomSnippets)
  const [search, setSearch] = useState('')

  const allSnippets = [...builtinSnippets, ...customSnippets]
  const filtered = search
    ? allSnippets.filter((s) => s.name.toLowerCase().includes(search.toLowerCase()) || s.sql.toLowerCase().includes(search.toLowerCase()))
    : allSnippets

  // Group by category
  const categories = new Map<string, Snippet[]>()
  for (const s of filtered) {
    const list = categories.get(s.category) || []
    list.push(s)
    categories.set(s.category, list)
  }

  const handleInsert = (sql: string) => {
    setRawText(sql)
  }

  const handleCopy = (sql: string) => {
    navigator.clipboard.writeText(sql)
  }

  const handleAdd = () => {
    const sql = useSqlStore.getState().rawText.trim()
    if (!sql) return
    const name = prompt('输入片段名称:')
    if (!name) return
    const category = prompt('输入分类 (留空为"自定义"):') || '自定义'
    const snippet: Snippet = {
      id: 'custom-' + Date.now(),
      name,
      sql,
      category,
      builtin: false,
    }
    const newSnippets = [snippet, ...customSnippets]
    setCustomSnippets(newSnippets)
    saveCustomSnippets(newSnippets)
  }

  const handleDelete = (id: string) => {
    const newSnippets = customSnippets.filter((s) => s.id !== id)
    setCustomSnippets(newSnippets)
    saveCustomSnippets(newSnippets)
  }

  const collapseItems = Array.from(categories.entries()).map(([cat, items]) => ({
    key: cat,
    label: (
      <span style={{ fontSize: 12, fontWeight: 600 }}>
        {cat} <Tag style={{ fontSize: 10, marginLeft: 4 }}>{items.length}</Tag>
      </span>
    ),
    children: (
      <List
        size="small"
        dataSource={items}
        renderItem={(snippet) => (
          <List.Item
            style={{ padding: '6px 0', cursor: 'pointer' }}
            actions={[
              <Tooltip key="copy" title="复制">
                <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => handleCopy(snippet.sql)} />
              </Tooltip>,
              ...(!snippet.builtin ? [
                <Tooltip key="del" title="删除">
                  <Button type="text" size="small" icon={<DeleteOutlined />} danger onClick={() => handleDelete(snippet.id)} />
                </Tooltip>,
              ] : []),
            ]}
            onClick={() => handleInsert(snippet.sql)}
          >
            <List.Item.Meta
              avatar={<CodeOutlined style={{ color: '#1677ff', fontSize: 14 }} />}
              title={<span style={{ fontSize: 12 }}>{snippet.name}</span>}
              description={
                <code style={{ fontSize: 10, color: '#999', fontFamily: 'var(--font-mono)' }}>
                  {snippet.sql.replace(/\s+/g, ' ').slice(0, 60)}...
                </code>
              }
            />
          </List.Item>
        )}
      />
    ),
  }))

  return (
    <div style={{ padding: 12, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12, gap: 8 }}>
        <BookOutlined style={{ color: '#722ed1' }} />
        <span style={{ fontWeight: 600, fontSize: 14 }}>代码片段</span>
        <Tag>{allSnippets.length}</Tag>
        <div style={{ flex: 1 }} />
        <Tooltip title="将当前编辑器内容保存为片段">
          <Button size="small" icon={<PlusOutlined />} onClick={handleAdd}>保存</Button>
        </Tooltip>
      </div>

      <Input.Search
        placeholder="搜索片段..."
        size="small"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        allowClear
        style={{ marginBottom: 12 }}
      />

      <div style={{ flex: 1, overflow: 'auto' }}>
        {filtered.length === 0 ? (
          <Empty description="没有匹配的片段" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 40 }} />
        ) : (
          <Collapse
            size="small"
            defaultActiveKey={Array.from(categories.keys())}
            items={collapseItems}
            ghost
          />
        )}
      </div>
    </div>
  )
}
