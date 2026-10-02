export interface UIButtonProps {
  readonly type?: 'button' | 'submit';
  readonly disabled?: boolean;
  readonly busy?: boolean;
  readonly variant?: 'secondary' | 'primary' | 'ghost' | 'danger' | 'row';
  readonly size?: 'default' | 'compact';
}

export interface UIToggleButtonProps extends Omit<UIButtonProps, 'type'> {
  readonly modelValue: boolean;
}

export interface UISubmitButtonProps extends Omit<UIButtonProps, 'type'> {}

export interface UIFormProps {
  readonly submit: () => void | Promise<void>;
}

export interface UIPaginationProps {
  readonly label: string;
  readonly total: number;
  readonly modelValue: number;
  readonly pageSize: number;
  readonly disabled?: boolean;
}

export interface UILinkProps {
  readonly iconOnly?: boolean;
  readonly href: string;
  readonly newTab?: boolean;
  readonly variant?: 'secondary' | 'primary' | 'ghost';
  readonly size?: 'default' | 'compact';
}

export interface UIIconButtonProps {
  readonly label: string;
  readonly icon: 'refresh' | 'download' | 'back' | 'forward';
  readonly busy?: boolean;
  readonly disabled?: boolean;
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

export interface UITabsProps {
  readonly label: string;
  readonly modelValue: string;
  readonly items: readonly { value: string; label: string }[];
}

export interface UIWebFrameProps {
  readonly src: string;
  readonly title: string;
  readonly target?: string;
}

export interface UIWebNavigation {
  readonly url: string;
  readonly canBack: boolean;
  readonly canForward: boolean;
  readonly busy: boolean;
  readonly error: string;
}

export interface UIWebFrameHandle {
  back(): void;
  forward(): void;
  reload(): void;
  navigate(url: string): void;
}

export interface UIMarkdownProps {
  readonly source: string;
  readonly headingOffset?: 1 | 2 | 3;
}

export interface UIRevealProps {
  readonly label: string;
  readonly modelValue: boolean;
}

export interface UIFileProps {
  readonly name: string;
  readonly size: number;
  readonly modelValue: boolean;
  readonly busy?: boolean;
  readonly disabled?: boolean;
}

export interface UICodeProps {
  readonly source: string;
  readonly label: string;
  readonly language?: string;
}

export interface UIConnectionStatusProps {
  readonly state: 'connected' | 'connecting' | 'disconnected' | 'error';
}

export interface UITerminalControlsProps extends UIConnectionStatusProps {
  readonly disabled?: boolean;
  readonly busy?: boolean;
}

export interface UISwitchProps {
  readonly label: string;
  readonly modelValue: boolean;
  readonly disabled?: boolean;
  readonly busy?: boolean;
  readonly compact?: boolean;
}

export interface UIStatusProps {
  readonly tone?: 'muted' | 'info' | 'success' | 'danger';
}

export interface UIBadgeProps {
  readonly tone?: 'accent' | 'quartz' | 'emerald' | 'sapphire' | 'amethyst' | 'ruby';
}

export interface UIDifficultyBadgeProps {
  readonly level: 1 | 2 | 3 | 4 | 5;
}

export interface UITextFieldProps {
  readonly id: string;
  readonly label: string;
  readonly modelValue: string;
  readonly disabled?: boolean;
  readonly required?: boolean;
  readonly labelHidden?: boolean;
  readonly size?: 'default' | 'compact';
  readonly readonly?: boolean;
  readonly tone?: 'default' | 'success' | 'danger';
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
