import { useMemo, useCallback, useRef } from 'react'
import { Button, Space, Tooltip } from 'antd'
import { DownloadOutlined, ExpandOutlined, AimOutlined } from '@ant-design/icons'
import { ReactFlow, Background, Controls, MiniMap, useReactFlow, ReactFlowProvider, BackgroundVariant } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { toPng } from 'html-to-image'
import { saveAs } from 'file-saver'
import { useSqlStore } from '@/store/useSqlStore'
import TableNode from './TableNode'
import JoinEdge from './JoinEdge'
import type { GraphNode, GraphEdge } from '@/types/sql'

const nodeTypes = { tableNode: TableNode }
const edgeTypes = { joinEdge: JoinEdge }

function RelationGraphInner() {
  const result = useSqlStore((s) => s.result)
  const containerRef = useRef<HTMLDivElement>(null)
  const { fitView } = useReactFlow()

  const { nodes, edges } = useMemo(() => {
    if (!result?.graph) return { nodes: [], edges: [] }

    const flowNodes = (result.graph.nodes || []).map((n: GraphNode) => {
      const d = (n.data || {}) as Record<string, unknown>
      return {
        id: n.id,
        type: n.type || 'tableNode',
        position: n.position || { x: 0, y: 0 },
        data: {
          tableName: d.tableName || '',
          alias: d.alias || '',
          role: d.role || '',
          selectedFields: Array.isArray(d.selectedFields) ? d.selectedFields : [],
          filterFields: Array.isArray(d.filterFields) ? d.filterFields : [],
          joinFields: Array.isArray(d.joinFields) ? d.joinFields : [],
        },
      }
    })

    const nodeIds = new Set(flowNodes.map((n) => n.id))
    const flowEdges = (result.graph.edges || [])
      .filter((e: GraphEdge) => e.source && e.target && nodeIds.has(e.source) && nodeIds.has(e.target))
      .map((e: GraphEdge) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        type: e.type || 'joinEdge',
        label: e.label || '',
        data: e.data || {},
      }))

    return { nodes: flowNodes, edges: flowEdges }
  }, [result])

  const handleExportPNG = useCallback(async () => {
    if (!containerRef.current) return
    try {
      const dataUrl = await toPng(containerRef.current, {
        backgroundColor: '#ffffff',
        pixelRatio: 2,
      })
      saveAs(dataUrl, 'sql-lens-graph.png')
    } catch {
      // ignore
    }
  }, [])

  const handleFitView = useCallback(() => {
    fitView({ duration: 400, padding: 0.3 })
  }, [fitView])

  if (nodes.length === 0) {
    return (
      <div style={{ textAlign: 'center', padding: 40, color: '#999', fontSize: 13 }}>
        暂无表关系数据
      </div>
    )
  }

  return (
    <div ref={containerRef} style={{ width: '100%', height: 480, position: 'relative', borderRadius: 8, overflow: 'hidden', border: '1px solid #f0f0f0' }}>
      <div style={{ position: 'absolute', top: 10, right: 10, zIndex: 10 }}>
        <Space size={6}>
          <Tooltip title="适应画布">
            <Button size="small" icon={<ExpandOutlined />} onClick={handleFitView} style={{ borderRadius: 6 }} />
          </Tooltip>
          <Tooltip title="导出 PNG">
            <Button size="small" icon={<DownloadOutlined />} onClick={handleExportPNG} style={{ borderRadius: 6 }} />
          </Tooltip>
        </Space>
      </div>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        minZoom={0.3}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#e8e8e8" />
        <Controls style={{ borderRadius: 6 }} />
        <MiniMap
          nodeStrokeWidth={3}
          pannable
          zoomable
          style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          maskColor="rgba(0,0,0,0.08)"
        />
      </ReactFlow>
    </div>
  )
}

export default function RelationGraph() {
  return (
    <ReactFlowProvider>
      <RelationGraphInner />
    </ReactFlowProvider>
  )
}
