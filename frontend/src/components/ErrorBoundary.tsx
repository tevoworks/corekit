import { Component, type ReactNode, type ErrorInfo } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    if (import.meta.env.DEV) {
      console.error('ErrorBoundary caught:', error, info)
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="h-full flex items-center justify-center p-8">
          <div className="text-center">
            <h1 className="text-xl font-bold text-[var(--on-surface)] mb-2">Something went wrong</h1>
            <p className="text-sm text-[var(--on-surface-variant)] mb-4">An unexpected error occurred. Please try again.</p>
            <button
              onClick={this.handleRetry}
              className="px-4 py-2 rounded-lg bg-[var(--primary)] text-white text-sm"
              data-testid="error-boundary-retry-button"
            >
              Retry
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
