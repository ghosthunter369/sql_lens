import ErrorBoundary from '@/components/ErrorBoundary'
import Home from '@/pages/Home'

export default function App() {
  return (
    <ErrorBoundary>
      <Home />
    </ErrorBoundary>
  )
}
