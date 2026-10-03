export type ProblemKind = 'file' | 'service';
export type ProblemTool = 'web' | 'files' | 'terminal';
export type Difficulty = 1 | 2 | 3 | 4 | 5;

export interface Concept {
  readonly id: string;
  readonly title: string;
  readonly requires: readonly string[];
  readonly related: readonly string[];
}

export interface Learning {
  readonly requires: readonly Concept[];
  readonly teaches: readonly Concept[];
}

export interface Problem {
  readonly slug: string;
  readonly title: string;
  readonly category: string;
  readonly kind: ProblemKind;
  readonly difficulty?: Difficulty;
  readonly solvedAt?: string;
  readonly cli?: readonly string[];
  readonly learning?: Learning;
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

export interface Connections {
  readonly before: readonly Problem[];
  readonly after: readonly Problem[];
  readonly related: readonly Problem[];
}

// Learning dependencies come from declared concepts; peer practice is separate.
export function connections(problem: Problem, catalog: readonly Problem[]): Connections {
  const before: Problem[] = [], after: Problem[] = [], related: Problem[] = [];
  const needs = new Set(problem.learning?.requires.map(item => item.id));
  const teaches = new Set(problem.learning?.teaches.map(item => item.id));
  const reading = new Set(problem.learning?.teaches.flatMap(item => item.related));
  for (const other of catalog) {
    if (other.slug === problem.slug || !other.learning) continue;
    const prior = other.learning.teaches.some(item => needs.has(item.id));
    const next = other.learning.requires.some(item => teaches.has(item.id));
    if (prior) before.push(other);
    if (next) after.push(other);
    if (!prior && !next && other.learning.teaches.some(item => teaches.has(item.id) || reading.has(item.id) || item.related.some(id => teaches.has(id)))) related.push(other);
  }
  const order = (a: Problem, b: Problem) => a.title.localeCompare(b.title, 'ko') || a.slug.localeCompare(b.slug);
  return { before: before.toSorted(order), after: after.toSorted(order), related: related.toSorted(order) };
}
