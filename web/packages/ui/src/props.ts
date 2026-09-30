export interface UIButtonProps {
  readonly type?: 'button' | 'submit';
  readonly disabled?: boolean;
  readonly busy?: boolean;
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
