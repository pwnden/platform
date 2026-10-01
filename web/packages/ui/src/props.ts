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
  readonly headingLevel?: 2 | 3;
}

export interface UISplitProps {
  readonly label: string;
  readonly modelValue: number;
  readonly min?: number;
  readonly max?: number;
}

export interface UIMarkdownProps {
  readonly source: string;
  readonly headingOffset?: 1 | 2 | 3;
}

export interface UIRevealProps {
  readonly label: string;
  readonly modelValue: boolean;
}

export interface UICodeProps {
  readonly source: string;
  readonly label: string;
}

export interface UIConnectionStatusProps {
  readonly state: 'connected' | 'connecting' | 'disconnected' | 'error';
}

export interface UITerminalControlsProps extends UIConnectionStatusProps {
  readonly disabled?: boolean;
  readonly busy?: boolean;
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

export interface UISelectProps {
  readonly id: string;
  readonly label: string;
  readonly modelValue: string;
  readonly options: readonly { value: string; label: string }[];
  readonly disabled?: boolean;
}

export interface UITerminalProps { readonly label: string; readonly enabled: boolean }
export interface UITerminalHandle {
  write(data: Uint8Array, rendered: () => void): void;
  clear(): void;
  focus(): void;
}
