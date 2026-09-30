export interface Endpoint {
  readonly name: string;
  readonly url: string;
}

export interface Run {
  readonly slug: string;
  readonly kind: 'file' | 'service';
  readonly fileCount: number;
  readonly endpoints: readonly Endpoint[];
}

export interface Submission {
  readonly slug: string;
  readonly accepted: boolean;
}

export interface Player {
  run(slug: string): Promise<Run>;
  stop(slug: string): Promise<void>;
  submit(slug: string, flag: string): Promise<Submission>;
}
