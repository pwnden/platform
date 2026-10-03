import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import VueRouter from 'vue-router/vite';

const plugins = () => [VueRouter({ routesFolder: new URL('./apps/player/src/pages', import.meta.url).pathname, dts: false }), vue()];
export default defineConfig({
  test: {
    projects: [
      { extends: true, plugins: plugins(), test: {
        name: 'unit', include: ['tests/**/*.test.ts'], exclude: ['tests/features.test.ts', 'tests/controls.test.ts'], environment: 'node',
      } },
      { extends: true, plugins: plugins(), test: {
        name: 'features', include: ['tests/features.test.ts'], environment: './tests/environment.mjs',
        server: { deps: { inline: ['vue', /^@vue\//] } },
      } },
      { extends: true, plugins: plugins(), test: {
        name: 'controls', include: ['tests/controls.test.ts'], environment: 'happy-dom',
        server: { deps: { inline: ['vue', /^@vue\//] } },
      } },
    ],
  },
});
