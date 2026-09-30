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
