import { useRef } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import { useSqlStore } from '@/store/useSqlStore'
import { formatSQL } from '@/utils/format'

export default function SqlEditor() {
  const rawText = useSqlStore((s) => s.rawText)
  const setRawText = useSqlStore((s) => s.setRawText)
  const parseSql = useSqlStore((s) => s.parseSql)
  const dialect = useSqlStore((s) => s.dialect)
  const holderRef = useRef<HTMLDivElement>(null)

  const handleMount: OnMount = (editor, monaco) => {
    editor.addAction({
      id: 'parse-sql',
      label: 'Parse SQL',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
      run: () => parseSql(),
    })
    editor.addAction({
      id: 'format-sql',
      label: 'Format SQL',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF],
      run: () => {
        const currentText = editor.getValue()
        if (currentText.trim()) {
          const formatted = formatSQL(currentText, dialect)
          editor.setValue(formatted)
        }
      },
    })
  }

  return (
    <div ref={holderRef} style={{ flex: 1, minHeight: 0, height: '100%' }}>
      <Editor
        height="100%"
        defaultLanguage="sql"
        value={rawText}
        onChange={(value) => setRawText(value || '')}
        theme="vs-dark"
        options={{
          minimap: { enabled: false },
          fontSize: 14,
          fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', 'JetBrains Mono', Consolas, monospace",
          fontLigatures: true,
          wordWrap: 'on',
          lineNumbers: 'on',
          automaticLayout: true,
          scrollBeyondLastLine: false,
          lineNumbersMinChars: 4,
          glyphMargin: false,
          folding: true,
          lineDecorationsWidth: 8,
          padding: { top: 12, bottom: 12 },
          bracketPairColorization: { enabled: true },
          matchBrackets: 'always',
          renderLineHighlight: 'line',
          cursorBlinking: 'smooth',
          smoothScrolling: true,
          overviewRulerBorder: false,
          hideCursorInOverviewRuler: true,
          guides: { indentation: true, bracketPairs: true },
          tabSize: 2,
        }}
        onMount={handleMount}
        loading={
          <div style={{
            height: '100%',
            background: '#1a1a2e',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <span style={{ color: '#6b7280', fontSize: 13 }}>加载编辑器...</span>
          </div>
        }
      />
    </div>
  )
}
