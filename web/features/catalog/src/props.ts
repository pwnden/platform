export interface ProblemFilesHandle {
  readonly busy: boolean;
  readonly filename: string;
  download(): Promise<void>;
}
