// Bridge to Go methods bound by Wails (backend/desktop/bindings).
// The POS talks to the central API from Go, never directly from the WebView,
// so terminal credentials never reach JavaScript.

export interface TerminalInfo {
  version: string;
  api_base_url: string;
  activated: boolean;
}

export interface ServerStatus {
  reachable: boolean;
  status: string;
  version: string;
  latency_ms: number;
  error?: string;
}

interface DesktopApp {
  Info(): Promise<TerminalInfo>;
  CheckServer(): Promise<ServerStatus>;
}

declare global {
  interface Window {
    go?: { bindings?: { App?: DesktopApp } };
  }
}

/** Returns the Go bridge, or null when running in a plain browser (vite dev). */
export function desktop(): DesktopApp | null {
  return window.go?.bindings?.App ?? null;
}
