import { saveAs } from 'file-saver'
import type { SQLAnalysisResult } from '@/types/sql'
import { buildMarkdownReport } from '@/api/sql'

export async function exportMarkdown(result: SQLAnalysisResult) {
  const res = await buildMarkdownReport(result)
  if (res.success && res.data) {
    const blob = new Blob([res.data.markdown], { type: 'text/markdown;charset=utf-8' })
    saveAs(blob, `sql-analysis-${Date.now()}.md`)
  }
}

export async function exportJSON(result: SQLAnalysisResult) {
  const json = JSON.stringify(result, null, 2)
  const blob = new Blob([json], { type: 'application/json;charset=utf-8' })
  saveAs(blob, `sql-analysis-${Date.now()}.json`)
}

export function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text)
}
