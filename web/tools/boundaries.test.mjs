import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { checkBoundaries } from './boundaries.mjs';

const root = fileURLToPath(new URL('..', import.meta.url));

function fixture(run) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'pwnden-boundaries-'));
  try {
    for (const folder of ['apps', 'packages', 'domains', 'features']) {
      fs.cpSync(path.join(root, folder), path.join(directory, folder), { recursive: true, filter: file => !file.split(path.sep).includes('node_modules') });
    }
    fs.copyFileSync(path.join(root, 'tsconfig.base.json'), path.join(directory, 'tsconfig.base.json'));
    fs.copyFileSync(path.join(root, 'package.json'), path.join(directory, 'package.json'));
    run(directory);
  } finally { fs.rmSync(directory, { recursive: true, force: true }); }
}

test('the actual workspace follows the policy', () => assert.deepEqual(checkBoundaries(root), []));

for (const [name, file, content, expected] of [
  ['type import', 'domains/catalog/src/invalid.ts', "import type { APIClient } from '@pwnden/api';", /not allowed/],
  ['re-export', 'domains/play/src/invalid.ts', "export * from '@pwnden/catalog-feature';", /not allowed/],
  ['dynamic import', 'features/catalog/src/invalid.ts', "import('@pwnden/play-feature');", /not allowed/],
  ['type expression', 'domains/play/src/invalid.ts', "type Client = import('@pwnden/api').APIClient;", /not allowed/],
  ['relative path', 'features/catalog/src/invalid.ts', "import '../../../packages/ui/src/props';", /must stay inside/],
  ['package internals', 'apps/player/src/invalid.ts', "import '@pwnden/ui/src/props';", /public package exports/],
  ['alias', 'domains/catalog/src/invalid.ts', "import '#api';", /not allowed/],
  ['Vue script', 'features/catalog/src/Invalid.vue', '<script setup lang="ts">import { createAPI } from "@pwnden/api";</script><template><div /></template>', /not allowed/],
  ['Sectile outside UI', 'apps/player/src/invalid.ts', "import '@sectile/vue/text';", /not allowed/],
  ['Lucide outside UI', 'features/play/src/invalid.ts', "import { Target } from '@lucide/vue';", /not allowed/],
  ['xterm outside UI', 'features/terminal/src/invalid.ts', "import '@xterm/xterm';", /not allowed/],
  ['Sectile public re-export', 'packages/ui/src/invalid.ts', "export { TextField } from '@sectile/vue/text';", /must stay private/],
  ['Lucide public re-export', 'packages/ui/src/invalid.ts', "export { Target } from '@lucide/vue';", /must stay private/],
  ['computed loading', 'apps/player/src/invalid.ts', 'const name = "vue"; import(name);', /literal/],
  ['mixed acronym', 'packages/api/src/invalid.ts', 'export interface HttpApiClient {}', /acronyms/],
  ['style import', 'features/catalog/src/invalid.css', '@import "@sectile/vue/text";', /not allowed/],
  ['ambient reference', 'domains/catalog/src/invalid.ts', '/// <reference lib="dom" />\nexport {};', /ambient references/],
  ['reference path', 'domains/catalog/src/invalid.ts', '/// <reference path="../../../packages/api/src/index.ts" />\nexport {};', /must stay inside/],
]) {
  test(`rejects ${name}`, () => fixture(directory => {
    fs.writeFileSync(path.join(directory, file), content);
    assert.match(checkBoundaries(directory).join('\n'), expected);
  }));
}

test('a manifest edit does not authorize a forbidden dependency and cycles are detected', () => fixture(directory => {
  const file = path.join(directory, 'domains/catalog/package.json');
  const manifest = JSON.parse(fs.readFileSync(file));
  manifest.dependencies = { '@pwnden/api': 'workspace:0.1.0' };
  fs.writeFileSync(file, JSON.stringify(manifest));
  const errors = checkBoundaries(directory).join('\n');
  assert.match(errors, /manifest dependency is not allowed/);
  assert.match(errors, /dependency cycle/);
}));

test('rejects ambient browser types and aliases in domain config', () => fixture(directory => {
  const file = path.join(directory, 'domains/catalog/tsconfig.json');
  const config = JSON.parse(fs.readFileSync(file));
  config.compilerOptions = { lib: ['ES2023', 'DOM'], paths: { '#api': ['../../packages/api/src/index.ts'] } };
  fs.writeFileSync(file, JSON.stringify(config));
  const errors = checkBoundaries(directory).join('\n');
  assert.match(errors, /ambient/);
  assert.match(errors, /public package exports/);
}));

test('rejects dependency ranges', () => fixture(directory => {
  const file = path.join(directory, 'packages/ui/package.json');
  const manifest = JSON.parse(fs.readFileSync(file));
  manifest.dependencies.vue = '^3.5.43';
  fs.writeFileSync(file, JSON.stringify(manifest));
  assert.match(checkBoundaries(directory).join('\n'), /exact version/);
}));

test('rejects a Vite alias that remaps an otherwise public import', () => fixture(directory => {
  fs.appendFileSync(path.join(directory, 'apps/player/vite.config.mjs'), '\nconst resolve = { alias: { "@pwnden/ui": "../../packages/api/src/index.ts" } };');
  assert.match(checkBoundaries(directory).join('\n'), /Vite aliases/);
}));

test('rejects a source symlink across package boundaries', () => fixture(directory => {
  fs.symlinkSync('../../../packages/api/src/index.ts', path.join(directory, 'domains/catalog/src/invalid.ts'));
  assert.match(checkBoundaries(directory).join('\n'), /source symlinks/);
}));

test('rejects internal module cycles', () => fixture(directory => {
  fs.writeFileSync(path.join(directory, 'domains/catalog/src/a.ts'), "import './b';");
  fs.writeFileSync(path.join(directory, 'domains/catalog/src/b.ts'), "import './a';");
  assert.match(checkBoundaries(directory).join('\n'), /module dependency cycle/);
}));

test('root dependencies use exact versions and leave Sectile to UI', () => fixture(directory => {
  const file = path.join(directory, 'package.json');
  const manifest = JSON.parse(fs.readFileSync(file));
  manifest.devDependencies.vue = '^3.5.43';
  manifest.devDependencies['@sectile/vue'] = '0.18.2';
  fs.writeFileSync(file, JSON.stringify(manifest));
  const errors = checkBoundaries(directory).join('\n');
  assert.match(errors, /exact version/);
  assert.match(errors, /belong to the UI/);
}));
