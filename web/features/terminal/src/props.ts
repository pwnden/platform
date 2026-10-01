export interface TerminalPanelHandle {
  readonly busy: boolean;
  refresh(): Promise<void>;
}
