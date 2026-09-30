import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import fontLicenses from '@pwnden/ui/licenses' with { type: 'json' };

export default defineConfig({
  plugins: [vue(), {
    name: 'bundled-font-licenses',
    generateBundle() {
      for (const [name, source] of Object.entries(fontLicenses)) {
        this.emitFile({ type: 'asset', fileName: `assets/licenses/${name}-OFL.txt`, source });
      }
    },
  }],
  build: { outDir: '../../dist/player', emptyOutDir: true },
});
