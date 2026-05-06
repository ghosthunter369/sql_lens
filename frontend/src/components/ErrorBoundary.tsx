import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div style={{ padding: 24, textAlign: 'center', color: '#ff4d4f' }}>
          <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 8 }}>渲染出错</div>
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>
            {this.state.error?.message || '未知错误'}
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
