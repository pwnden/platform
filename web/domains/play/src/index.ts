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
  browser(slug: string, name: string): Promise<BrowserSession>;
  status(slug: string): Promise<RunStatus>;
  run(slug: string): Promise<Run>;
  stop(slug: string): Promise<void>;
  submit(slug: string, flag: string): Promise<Submission>;
}

export interface BrowserSession {
  readonly url: string;
  readonly target: string;
}

export interface RunStatus {
  readonly slug: string;
  readonly kind: 'file' | 'service';
  readonly state: 'ready' | 'stopped' | 'running' | 'unavailable';
  readonly endpoints: readonly Endpoint[];
}
