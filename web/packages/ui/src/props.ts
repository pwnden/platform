export interface UIButtonProps {
  readonly type?: 'button' | 'submit';
  readonly disabled?: boolean;
  readonly busy?: boolean;
  readonly variant?: 'secondary' | 'primary' | 'ghost' | 'danger' | 'row';
  readonly size?: 'default' | 'compact';
}

export interface UIPanelProps {
  readonly title: string;
  readonly headingID: string;
}

export interface UIMarkdownProps {
  readonly source: string;
}

export interface UIStatusProps {
  readonly tone?: 'muted' | 'info' | 'success' | 'danger';
}

export interface UITextFieldProps {
  readonly id: string;
  readonly label: string;
  readonly modelValue: string;
  readonly disabled?: boolean;
  readonly required?: boolean;
}

export interface UITerminalProps { readonly label: string; readonly enabled: boolean }
export interface UITerminalHandle {
  write(data: Uint8Array, rendered: () => void): void;
  clear(): void;
  focus(): void;
}
