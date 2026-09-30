import type { Catalog, Problem } from '@pwnden/catalog';
import type { Player, Run, Submission } from '@pwnden/play';

export class APIError extends Error {
  constructor(readonly code: string, readonly status: number) {
    super('The player request failed.');
    this.name = 'APIError';
  }
}

export interface APIClient {
  readonly catalog: Catalog;
  readonly player: Player;
}

export interface APIOptions {
  readonly token: string;
  readonly fetch?: typeof globalThis.fetch;
}

function object(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new APIError('invalid_response', 0);
  }
  return value as Record<string, unknown>;
}

function string(value: unknown): string {
  if (typeof value !== 'string') throw new APIError('invalid_response', 0);
  return value;
}

function kind(value: unknown): 'file' | 'service' {
  if (value !== 'file' && value !== 'service') throw new APIError('invalid_response', 0);
  return value;
}

function array(value: unknown): unknown[] {
  if (!Array.isArray(value)) throw new APIError('invalid_response', 0);
  return value;
}

function route(slug: string): string {
  if (slug.length > 40 || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
    throw new APIError('invalid_argument', 0);
  }
  return `/problems/${slug}`;
}

// The fixed relative base keeps bearer credentials on the server's own origin.
export function createAPI(options: APIOptions): APIClient {
  if (!/^[a-f0-9]{64}$/.test(options.token)) throw new APIError('unauthorized', 0);
  const requestFetch = options.fetch ?? globalThis.fetch.bind(globalThis);

  async function request(path: string, method = 'GET', body?: string): Promise<Record<string, unknown>> {
    let response: Response;
    try {
      response = await requestFetch(`/api/v1${path}`, {
        method,
        credentials: 'omit',
        cache: 'no-store',
        redirect: 'error',
        headers: {
          Authorization: `Bearer ${options.token}`,
          ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        },
        ...(body === undefined ? {} : { body }),
      });
    } catch {
      throw new APIError('network_error', 0);
    }
    let payload: Record<string, unknown>;
    try {
      payload = object(await response.json());
    } catch {
      throw new APIError('invalid_response', response.status);
    }
    if (!response.ok) {
      const error = object(payload.error);
      throw new APIError(string(error.code), response.status);
    }
    return payload;
  }

  return {
    catalog: {
      async list(): Promise<readonly Problem[]> {
        const payload = await request('/problems');
        return array(payload.problems).map(value => {
          const item = object(value);
          return { slug: string(item.slug), title: string(item.title), category: string(item.category), kind: kind(item.kind) };
        });
      },
    },
    player: {
      async run(slug): Promise<Run> {
        const item = await request(`${route(slug)}/run`, 'POST');
        if (!Number.isSafeInteger(item.file_count) || (item.file_count as number) < 0 || item.slug !== slug) {
          throw new APIError('invalid_response', 0);
        }
        return {
          slug, kind: kind(item.kind), fileCount: item.file_count as number,
          endpoints: array(item.endpoints).map(value => {
            const endpoint = object(value);
            const url = string(endpoint.url);
            if (!/^(http|tcp):\/\//.test(url)) throw new APIError('invalid_response', 0);
            return { name: string(endpoint.name), url };
          }),
        };
      },
      async stop(slug): Promise<void> {
        const item = await request(`${route(slug)}/run`, 'DELETE');
        if (item.slug !== slug) throw new APIError('invalid_response', 0);
      },
      async submit(slug, flag): Promise<Submission> {
        const path = `${route(slug)}/submissions`;
        const body = JSON.stringify({ flag });
        if (!flag.trim() || new TextEncoder().encode(body).byteLength > 4096) {
          throw new APIError('invalid_argument', 0);
        }
        const item = await request(path, 'POST', body);
        if (typeof item.accepted !== 'boolean' || item.slug !== slug) throw new APIError('invalid_response', 0);
        return { slug, accepted: item.accepted };
      },
    },
  };
}
