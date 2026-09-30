import type { Catalog, Problem, ProblemDetail } from '@pwnden/catalog';
import type { Endpoint, Player, Run, RunStatus, Submission } from '@pwnden/play';

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

function size(value: unknown): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) {
    throw new APIError('invalid_response', 0);
  }
  return value;
}

function endpoints(value: unknown): readonly Endpoint[] {
  return array(value).map(value => {
    const endpoint = object(value);
    const url = string(endpoint.url);
    if (!/^(http|tcp):\/\//.test(url)) throw new APIError('invalid_response', 0);
    return { name: string(endpoint.name), url };
  });
}

function fileID(id: string): string {
  if (!/^[a-f0-9]{64}$/.test(id)) throw new APIError('invalid_argument', 0);
  return id;
}

// The fixed relative base keeps bearer credentials on the server's own origin.
export function createAPI(options: APIOptions): APIClient {
  if (!/^[a-f0-9]{64}$/.test(options.token)) throw new APIError('unauthorized', 0);
  const requestFetch = options.fetch ?? globalThis.fetch.bind(globalThis);

  async function fetchResponse(path: string, method = 'GET', body?: string): Promise<Response> {
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
    if (!response.ok) {
      let payload: Record<string, unknown>;
      try { payload = object(await response.json()); }
      catch { throw new APIError('invalid_response', response.status); }
      const error = object(payload.error);
      throw new APIError(string(error.code), response.status);
    }
    return response;
  }

  async function request(path: string, method = 'GET', body?: string): Promise<Record<string, unknown>> {
    const response = await fetchResponse(path, method, body);
    try {
      return object(await response.json());
    } catch {
      throw new APIError('invalid_response', response.status);
    }
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
      async detail(slug): Promise<ProblemDetail> {
        const item = await request(route(slug));
        if (item.slug !== slug) throw new APIError('invalid_response', 0);
        return {
          slug, title: string(item.title), category: string(item.category), kind: kind(item.kind),
          description: string(item.description),
          files: array(item.files).map(value => {
            const file = object(value);
            const id = string(file.id);
            if (!/^[a-f0-9]{64}$/.test(id)) throw new APIError('invalid_response', 0);
            return { id, name: string(file.name), size: size(file.size) };
          }),
        };
      },
      async download(slug, id): Promise<Uint8Array> {
        const response = await fetchResponse(`${route(slug)}/files/${fileID(id)}`);
        if (response.headers.get('Content-Type') !== 'application/octet-stream') {
          throw new APIError('invalid_response', response.status);
        }
        try { return new Uint8Array(await response.arrayBuffer()); }
        catch { throw new APIError('network_error', 0); }
      },
    },
    player: {
      async status(slug): Promise<RunStatus> {
        const item = await request(`${route(slug)}/status`);
        const problemKind = kind(item.kind);
        if (item.slug !== slug || !['ready', 'stopped', 'running', 'unavailable'].includes(string(item.state)) ||
          (problemKind === 'file') !== (item.state === 'ready')) {
          throw new APIError('invalid_response', 0);
        }
        const addresses = endpoints(item.endpoints);
        if (item.state !== 'running' && addresses.length !== 0) throw new APIError('invalid_response', 0);
        return { slug, kind: problemKind, state: item.state as RunStatus['state'], endpoints: addresses };
      },
      async run(slug): Promise<Run> {
        const item = await request(`${route(slug)}/run`, 'POST');
        if (!Number.isSafeInteger(item.file_count) || (item.file_count as number) < 0 || item.slug !== slug) {
          throw new APIError('invalid_response', 0);
        }
        return {
          slug, kind: kind(item.kind), fileCount: item.file_count as number,
          endpoints: endpoints(item.endpoints),
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
