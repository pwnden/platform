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
