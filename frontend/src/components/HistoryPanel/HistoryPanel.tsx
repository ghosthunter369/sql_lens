import { useState } from 'react'
import { List, Tag, Button, Input, Popconfirm, Empty, Tooltip } from 'antd'
import {
  StarOutlined,
  StarFilled,
  DeleteOutlined,
  HistoryOutlined,
  CopyOutlined,
  ClearOutlined,
} from '@ant-design/icons'
import { useHistoryStore } from '@/store/useHistoryStore'
import { useSqlStore } from '@/store/useSqlStore'

const complexityColors: Record<string, string> = {
  LOW: 'green',
  MEDIUM: 'orange',
  HIGH: 'red',
}

export default function HistoryPanel() {
  const history = useHistoryStore((s) => s.history)
  const removeHistory = useHistoryStore((s) => s.removeHistory)
  const toggleStar = useHistoryStore((s) => s.toggleStar)
  const clearHistory = useHistoryStore((s) => s.clearHistory)
  const setRawText = useSqlStore((s) => s.setRawText)
  const [search, setSearch] = useState('')

  const filtered = history.filter((h) => {
    if (!search) return true
    const q = search.toLowerCase()
    return h.sql.toLowerCase().includes(q) || h.dialect.toLowerCase().includes(q)
  })

  const starred = filtered.filter((h) => h.starred)
  const recent = filtered.filter((h) => !h.starred)

  const formatTime = (ts: number) => {
    const d = new Date(ts)
    const now = new Date()
    const diff = now.getTime() - ts
    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
    if (diff < 604800000) return `${Math.floor(diff / 86400000)} 天前`
    return d.toLocaleDateString('zh-CN')
  }

  const handleLoad = (sql: string) => {
    setRawText(sql)
  }

  const handleCopy = (sql: string) => {
    navigator.clipboard.writeText(sql)
  }

  return (
    <div style={{ padding: 12, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12, gap: 8 }}>
        <HistoryOutlined style={{ color: '#1677ff' }} />
        <span style={{ fontWeight: 600, fontSize: 14 }}>历史记录</span>
        <Tag>{history.length}</Tag>
        <div style={{ flex: 1 }} />
        {history.length > 0 && (
          <Popconfirm title="清空所有历史记录？" onConfirm={clearHistory} okText="确定" cancelText="取消">
            <Button size="small" icon={<ClearOutlined />} danger>清空</Button>
          </Popconfirm>
        )}
      </div>

      <Input.Search
        placeholder="搜索 SQL 内容..."
        size="small"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        allowClear
        style={{ marginBottom: 12 }}
      />

      <div style={{ flex: 1, overflow: 'auto' }}>
        {filtered.length === 0 ? (
          <Empty
            description={search ? '没有匹配的记录' : '暂无历史记录'}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            style={{ marginTop: 40 }}
          />
        ) : (
          <>
            {starred.length > 0 && (
              <div style={{ marginBottom: 12 }}>
                <div style={{ fontSize: 11, color: '#999', marginBottom: 6, fontWeight: 600 }}>收藏</div>
                <List
                  size="small"
                  dataSource={starred}
                  renderItem={(item) => <HistoryItem item={item} onLoad={handleLoad} onCopy={handleCopy} onRemove={removeHistory} onToggleStar={toggleStar} formatTime={formatTime} />}
                />
              </div>
            )}
            {recent.length > 0 && (
              <div>
                <div style={{ fontSize: 11, color: '#999', marginBottom: 6, fontWeight: 600 }}>最近</div>
                <List
                  size="small"
                  dataSource={recent}
                  renderItem={(item) => <HistoryItem item={item} onLoad={handleLoad} onCopy={handleCopy} onRemove={removeHistory} onToggleStar={toggleStar} formatTime={formatTime} />}
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function HistoryItem({ item, onLoad, onCopy, onRemove, onToggleStar, formatTime }: {
  item: import('@/store/useHistoryStore').HistoryItem
  onLoad: (sql: string) => void
  onCopy: (sql: string) => void
  onRemove: (id: string) => void
  onToggleStar: (id: string) => void
  formatTime: (ts: number) => string
}) {
  const preview = item.sql.replace(/\s+/g, ' ').slice(0, 120)

  return (
    <List.Item
      style={{
        padding: '8px 10px',
        cursor: 'pointer',
        borderRadius: 6,
        marginBottom: 4,
        border: '1px solid #f0f0f0',
        background: '#fafafa',
      }}
      actions={[
        <Tooltip key="star" title={item.starred ? '取消收藏' : '收藏'}>
          <Button
            type="text"
            size="small"
            icon={item.starred ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />}
            onClick={(e) => { e.stopPropagation(); onToggleStar(item.id) }}
          />
        </Tooltip>,
        <Tooltip key="copy" title="复制 SQL">
          <Button
            type="text"
            size="small"
            icon={<CopyOutlined />}
            onClick={(e) => { e.stopPropagation(); onCopy(item.sql) }}
          />
        </Tooltip>,
        <Popconfirm key="del" title="删除此记录？" onConfirm={() => onRemove(item.id)} okText="确定" cancelText="取消">
          <Button type="text" size="small" icon={<DeleteOutlined />} danger onClick={(e) => e.stopPropagation()} />
        </Popconfirm>,
      ]}
      onClick={() => onLoad(item.sql)}
    >
      <List.Item.Meta
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <code style={{ fontSize: 11, fontFamily: 'var(--font-mono)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {preview}
            </code>
          </div>
        }
        description={
          <div style={{ display: 'flex', gap: 6, alignItems: 'center', fontSize: 11 }}>
            <Tag color="blue" style={{ fontSize: 10, padding: '0 4px', lineHeight: '16px' }}>{item.dialect}</Tag>
            <Tag color={complexityColors[item.summary.complexity] || 'default'} style={{ fontSize: 10, padding: '0 4px', lineHeight: '16px' }}>
              {item.summary.tableCount}表 {item.summary.joinCount}JOIN
            </Tag>
            <span style={{ color: '#999' }}>{formatTime(item.timestamp)}</span>
          </div>
        }
      />
    </List.Item>
  )
}
