import { defineConfig, searchForWorkspaceRoot } from 'vite';
import vue from '@vitejs/plugin-vue';
import fontLicenses from '@pwnden/ui/licenses' with { type: 'json' };

const styleNonce = process.env.PWNDEN_STYLE_NONCE;

export default defineConfig({
  cacheDir: '../../node_modules/.vite/player',
  html: { cspNonce: styleNonce },
  server: {
    host: '0.0.0.0', port: 5173, strictPort: true,
    cors: false,
    ws: { path: '/__vite_hmr', clientPort: 0 },
    watch: { usePolling: true, interval: 300 },
    fs: { allow: ['../..'] },
    forwardConsole: false,
  },
  plugins: [vue(), {
    name: 'workspace-source-watch',
    configureServer(server) {
      // Type-only SFC dependencies do not enter Vite's runtime module graph.
      // Watch their workspace directories so Vue can invalidate its type cache.
      const workspaceRoot = searchForWorkspaceRoot(server.config.root);
      server.watcher.add(['domains', 'features', 'packages'].map(name => `${workspaceRoot}/${name}`));
    },
  }, {
    name: 'player-style-nonce',
    'transformIndexHtml'() {
      return styleNonce ? [{ tag: 'meta', attrs: { name: 'pwnden-style-nonce', content: styleNonce }, injectTo: 'head' }] : [];
    },
  }, {
    name: 'bundled-font-licenses',
    generateBundle() {
      for (const [name, source] of Object.entries(fontLicenses)) {
        this.emitFile({ type: 'asset', fileName: `assets/licenses/${name}-OFL.txt`, source });
      }
    },
  }],
  build: { outDir: '../../dist/player', emptyOutDir: true },
});
