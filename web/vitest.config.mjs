import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  test: {
    projects: [
      { extends: true, test: {
        name: 'unit', include: ['tests/**/*.test.ts'], exclude: ['tests/features.test.ts', 'tests/controls.test.ts'], environment: 'node',
      } },
      { extends: true, test: {
        name: 'features', include: ['tests/features.test.ts'], environment: './tests/environment.mjs',
        server: { deps: { inline: ['vue', /^@vue\//] } },
      } },
      { extends: true, test: {
        name: 'controls', include: ['tests/controls.test.ts'], environment: 'happy-dom',
        server: { deps: { inline: ['vue', /^@vue\//] } },
      } },
    ],
  },
});
