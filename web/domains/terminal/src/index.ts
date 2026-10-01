export interface TerminalSize { readonly cols: number; readonly rows: number }
export type TerminalEvent =
  | { readonly type: 'output'; readonly data: Uint8Array; acknowledge(): void }
  | { readonly type: 'exit'; readonly code: number }
  | { readonly type: 'error'; readonly code: string }
  | { readonly type: 'closed' };

export interface TerminalSession {
	readonly reused?: boolean;
  input(data: Uint8Array): void;
  resize(size: TerminalSize): void;
  close(): void;
}

// Closing a browser attachment preserves the environment until inactive expiry.
export interface TerminalConnection {
  readonly ready: Promise<TerminalSession>;
  close(): void;
}
export interface Terminals {
  connect(slug: string, size: TerminalSize, receive: (event: TerminalEvent) => void): TerminalConnection;
  list(): Promise<readonly RetainedEnvironment[]>;
  stop(slug: string): Promise<void>;
}

export interface RetainedEnvironment {
  readonly slug: string;
  readonly title: string;
  readonly connected: boolean;
  readonly expiresAt: string | null;
}
