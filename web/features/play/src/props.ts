export interface PlayPanelHandle {
  refresh(): Promise<void>;
}

export interface ProblemWebHandle {
  readonly url: string;
  reload(): void;
}
