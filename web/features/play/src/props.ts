import type { WorkspaceInfo } from '@pwnden/play';

export interface EnvironmentState {
  readonly preparing: boolean;
  readonly error: string;
  readonly retained: readonly WorkspaceInfo[];
  readonly ending: boolean;
  retry(): Promise<void>;
  stop(slug: string): Promise<void>;
}

export interface ProblemWebHandle {
  readonly url: string;
  readonly busy: boolean;
  readonly canBack: boolean;
  readonly canForward: boolean;
  readonly canReload: boolean;
  back(): void;
  forward(): void;
  reload(): void;
}
