export type ProblemKind = 'file' | 'service';
export type ProblemTool = 'web' | 'files' | 'terminal';
export type Difficulty = 1 | 2 | 3 | 4 | 5;

export interface Problem {
  readonly slug: string;
  readonly title: string;
  readonly category: string;
  readonly kind: ProblemKind;
  readonly difficulty?: Difficulty;
  readonly solvedAt?: string;
  readonly cli?: readonly string[];
}

export interface Catalog {
  list(): Promise<readonly Problem[]>;
  detail(slug: string): Promise<ProblemDetail>;
  download(slug: string, id: string, maxBytes?: number): Promise<Uint8Array>;
  guidance(slug: string, id: string): Promise<string>;
}

export interface ProblemFile {
  readonly id: string;
  readonly name: string;
  readonly size: number;
}

export interface ProblemDetail extends Problem {
  readonly answer?: string;
  readonly tools: readonly ProblemTool[];
  readonly description: string;
  readonly files: readonly ProblemFile[];
  readonly hintCount: number;
  readonly walkthrough: boolean;
}
