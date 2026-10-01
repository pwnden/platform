import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';
import { createServer, normalizePath } from 'vite';
import { expect, it } from 'vitest';

it('updates type-only workspace dependencies before compiling newly added Vue props', async () => {
  const sourceRoot = fileURLToPath(new URL('../packages/ui/src/', import.meta.url));
  const directory = await mkdtemp(join(sourceRoot, 'pwnden-hmr-'));
  const typePath = join(directory, 'props.ts');
  const existingPath = join(directory, 'Existing.vue');
  const addedPath = join(directory, 'Added.vue');
  const initial = 'export interface ExistingProps { label: string }\n';
  const component = (type: string) => `<script setup lang="ts">import type { ${type} } from './props'; defineProps<${type}>();</script><template><p>{{ label }}</p></template>`;
  let server: Awaited<ReturnType<typeof createServer>> | undefined;
  try {
    await writeFile(typePath, initial);
    await writeFile(existingPath, component('ExistingProps'));
    await writeFile(addedPath, component('AddedProps'));
    server = await createServer({
      root: fileURLToPath(new URL('../apps/player/', import.meta.url)),
      configFile: fileURLToPath(new URL('../apps/player/vite.config.mjs', import.meta.url)),
      configLoader: 'native',
      server: { host: '127.0.0.1', port: 0, strictPort: false },
      optimizeDeps: { noDiscovery: true, include: [] },
      logLevel: 'silent',
    });
    await server.listen();
    const origin = server.resolvedUrls!.local[0]!;
    const fetchModule = (path: string) => fetch(`${origin}@fs/${normalizePath(path)}`);
    expect((await fetchModule(existingPath)).status).toBe(200);
    // Seed the same missing-type failure observed in the running Docker server.
    expect((await fetchModule(addedPath)).status).toBe(500);
    await expect.poll(() => Object.entries(server!.watcher.getWatched()).some(([path, files]) =>
      normalizePath(path) === normalizePath(directory) && files.includes('props.ts'),
    ), { timeout: 5000 }).toBe(true);
    await writeFile(typePath, initial + 'export interface AddedProps { label: string; count?: number }\n');
    await expect.poll(async () => (await fetchModule(addedPath)).status, { timeout: 5000 }).toBe(200);
    const module = await (await fetchModule(addedPath)).text();
    expect(module).toContain('count:');
    expect(module).toContain('type: Number');
  } finally {
    await server?.close();
    await rm(directory, { recursive: true, force: true });
  }
}, 15000);
