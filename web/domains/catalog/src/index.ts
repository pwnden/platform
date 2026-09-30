export type ProblemKind = 'file' | 'service';

export interface Problem {
  readonly slug: string;
  readonly title: string;
  readonly category: string;
  readonly kind: ProblemKind;
}

export interface Catalog {
  list(): Promise<readonly Problem[]>;
}
