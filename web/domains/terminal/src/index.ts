export interface TerminalSize { readonly cols: number; readonly rows: number }
export type TerminalEvent =
  | { readonly type: 'output'; readonly data: Uint8Array; acknowledge(): void }
  | { readonly type: 'exit'; readonly code: number }
  | { readonly type: 'error'; readonly code: string }
  | { readonly type: 'closed' };

export interface TerminalSession {
  input(data: Uint8Array): void;
  resize(size: TerminalSize): void;
  close(): void;
}

// DOM cancellation and transport details stay in the adapter. Closing a
// connection attempt is the same ownership operation as closing a ready shell.
export interface TerminalConnection {
  readonly ready: Promise<TerminalSession>;
  close(): void;
}
export interface Terminals {
  connect(slug: string, size: TerminalSize, receive: (event: TerminalEvent) => void): TerminalConnection;
}
