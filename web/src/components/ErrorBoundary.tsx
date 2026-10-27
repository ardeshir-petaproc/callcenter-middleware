import { Component, type ErrorInfo, type ReactNode } from 'react';
import { AlertTriangle } from 'lucide-react';

interface State { error: Error | null }

/** Keeps a crash in one page from blanking the whole app shell. */
export class ErrorBoundary extends Component<{ children: ReactNode; resetKey?: string }, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State { return { error }; }

  componentDidCatch(error: Error, info: ErrorInfo) { console.error('UI error', error, info.componentStack); }

  componentDidUpdate(prev: { resetKey?: string }) {
    if (prev.resetKey !== this.props.resetKey && this.state.error) this.setState({ error: null });
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="card">
        <div className="empty">
          <AlertTriangle aria-hidden />
          <div className="title">Something went wrong on this page</div>
          <div className="small mono">{this.state.error.message}</div>
          <button className="btn" style={{ marginTop: 8 }} onClick={() => this.setState({ error: null })}>Try again</button>
        </div>
      </div>
    );
  }
}
