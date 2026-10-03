import type { Catalog, Problem, ProblemDetail, ProblemTool, Difficulty } from '@pwnden/catalog';
import type { Endpoint, Player, Run, RunStatus, Submission, Workspaces } from '@pwnden/play';
import type { Terminals } from '@pwnden/terminal';
import { connectTerminal } from './terminal';
import type { TerminalTransport } from './terminal';
import { connectWorkspace } from './workspace';

export class APIError extends Error {
  constructor(readonly code: string, readonly status: number) {
    super('The player request failed.');
    this.name = 'APIError';
  }
}

export interface APIClient {
  readonly catalog: Catalog;
  readonly player: Player;
  readonly terminals: Terminals;
  readonly workspaces: Workspaces;
}

export interface APIOptions {
  readonly token: string;
  readonly onUnauthorized?: () => void;
  readonly fetch?: typeof globalThis.fetch;
  readonly socket?: (url: string, protocol: string) => TerminalTransport;
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

function difficulty(value: unknown): { difficulty?: Difficulty } {
  if (value === undefined) return {};
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 1 || value > 5) throw new APIError('invalid_response', 0);
  return { difficulty: value as Difficulty };
}

function completion(value: unknown): { solvedAt?: string } {
  if (value === undefined) return {};
  if (typeof value !== 'string' || !Number.isFinite(Date.parse(value))) throw new APIError('invalid_response', 0);
  return { solvedAt: value };
}

function cli(value: unknown): { cli?: readonly string[] } {
  if (value === undefined) return {};
  const names = array(value).map(value => string(value));
  if (names.length > 32 || new Set(names).size !== names.length || names.some(name => !/^[a-z0-9][a-z0-9.+_-]{0,63}$/.test(name))) throw new APIError('invalid_response', 0);
  return { cli: names };
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
      if (response.status === 401) options.onUnauthorized?.();
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
    workspaces: {
      async list() {
        const payload = await request('/workspaces');
        return array(payload.workspaces).map(value => {
          const item = object(value);
          if (typeof item.connected !== 'boolean' || (item.expires_at !== null && typeof item.expires_at !== 'string')) throw new APIError('invalid_response', 0);
          return { slug: string(item.slug), title: string(item.title), connected: item.connected, expiresAt: item.expires_at as string | null };
        });
      },
      async stop(slug) {
        const item = await request(`${route(slug)}/run`, 'DELETE');
        if (item.slug !== slug) throw new APIError('invalid_response', 0);
      },
      connect(slug, failed) {
        return connectWorkspace(`/api/v1${route(slug)}/workspace`, options.token, requestFetch, code => {
          if (code === 'unauthorized') options.onUnauthorized?.();
          failed(code);
        });
      },
    },
    terminals: {
      async list() {
        const payload = await request('/workspaces');
        return array(payload.workspaces).map(value => {
          const item = object(value);
          if (typeof item.connected !== 'boolean' || (item.expires_at !== null && typeof item.expires_at !== 'string')) throw new APIError('invalid_response', 0);
          return { slug: string(item.slug), title: string(item.title), connected: item.connected, expiresAt: item.expires_at as string | null };
        });
      },
      async stop(slug) {
        const item = await request(`${route(slug)}/run`, 'DELETE');
        if (item.slug !== slug) throw new APIError('invalid_response', 0);
      },
      connect(slug, size, receive) {
        const path = `/api/v1${route(slug)}/terminal`;
        return connectTerminal(path, options.token, size, event => {
          if (event.type === 'error' && event.code === 'unauthorized') options.onUnauthorized?.();
          receive(event);
        },
          options.socket ?? ((url, protocol) => new WebSocket(url, protocol)), globalThis.location.origin);
      },
    },
    catalog: {
      async list(): Promise<readonly Problem[]> {
        const payload = await request('/problems');
        return array(payload.problems).map(value => {
          const item = object(value);
          return { slug: string(item.slug), title: string(item.title), category: string(item.category), kind: kind(item.kind), ...difficulty(item.difficulty), ...completion(item.solved_at),
            ...cli(item.cli) };
        });
      },
      async detail(slug): Promise<ProblemDetail> {
        const item = await request(route(slug));
        if (item.slug !== slug) throw new APIError('invalid_response', 0);
        const hintCount = size(item.hint_count);
        if (hintCount > 10 || typeof item.walkthrough !== 'boolean') throw new APIError('invalid_response', 0);
        return {
          slug, title: string(item.title), category: string(item.category), kind: kind(item.kind),
          ...difficulty(item.difficulty),
          ...cli(item.cli),
          ...completion(item.solved_at),
          ...(item.solved_at === undefined ? {} : { answer: string(item.answer) }),
          description: string(item.description),
          tools: array(item.tools).map(tool => {
            if (tool !== 'web' && tool !== 'files' && tool !== 'terminal') throw new APIError('invalid_response', 0);
            return tool as ProblemTool;
          }),
          hintCount, walkthrough: item.walkthrough,
          files: array(item.files).map(value => {
            const file = object(value);
            const id = string(file.id);
            if (!/^[a-f0-9]{64}$/.test(id)) throw new APIError('invalid_response', 0);
            return { id, name: string(file.name), size: size(file.size) };
          }),
        };
      },
      async guidance(slug, id): Promise<string> {
        if (!/^(walkthrough|hint-([1-9]|10))$/.test(id)) throw new APIError('invalid_argument', 0);
        const item = await request(`${route(slug)}/guidance/${id}`);
        if (item.id !== id) throw new APIError('invalid_response', 0);
        return string(item.content);
      },
      async download(slug, id, maxBytes): Promise<Uint8Array> {
        if (maxBytes !== undefined && (!Number.isSafeInteger(maxBytes) || maxBytes <= 0)) throw new APIError('invalid_argument', 0);
        const response = await fetchResponse(`${route(slug)}/files/${fileID(id)}`);
        if (response.headers.get('Content-Type') !== 'application/octet-stream') {
          throw new APIError('invalid_response', response.status);
        }
        try {
          if (maxBytes === undefined) return new Uint8Array(await response.arrayBuffer());
          const reader = response.body?.getReader();
          if (!reader) throw new APIError('invalid_response', 0);
          const chunks: Uint8Array[] = [];
          let length = 0;
          try {
            while (true) {
              const part = await reader.read();
              if (part.done) break;
              length += part.value.byteLength;
              if (length > maxBytes) throw new APIError('preview_too_large', 0);
              chunks.push(part.value);
            }
          } finally { await reader.cancel(); reader.releaseLock(); }
          const result = new Uint8Array(length);
          let offset = 0;
          for (const chunk of chunks) { result.set(chunk, offset); offset += chunk.length; }
          return result;
        } catch (error) {
          if (error instanceof APIError) throw error;
          throw new APIError('network_error', 0);
        }
      },
    },
    player: {
      async browser(slug, name) {
        const body = JSON.stringify({ name });
        if (!name.trim() || new TextEncoder().encode(body).byteLength > 4096) throw new APIError('invalid_argument', 0);
        const item = await request(`${route(slug)}/browser`, 'POST', body);
        const url = string(item.url);
        const target = string(item.target);
        // Wrapper and target share the problem ingress origin; the player is separate.
        const targetPort = /^http:\/\/127\.0\.0\.1:([1-9][0-9]{0,4})$/.exec(target)?.[1];
        const wrapperPort = /^http:\/\/127\.0\.0\.1:([1-9][0-9]{0,4})\/__pwnden_browser\/[a-f0-9]{64}$/.exec(url)?.[1];
        if (item.slug !== slug || item.name !== name || !targetPort || !wrapperPort ||
          Number(targetPort) < 1 || Number(targetPort) > 65535 || Number(wrapperPort) < 1 || Number(wrapperPort) > 65535 ||
          Number(targetPort) !== Number(wrapperPort) || target === globalThis.location?.origin) throw new APIError('invalid_response', 0);
        return { url, target };
      },
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
