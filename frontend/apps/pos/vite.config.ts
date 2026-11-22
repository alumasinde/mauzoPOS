import { svelte } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';

// The POS UI is embedded into the Wails .exe. Wails embeds files from inside
// its Go package, so the build is written straight into backend/desktop.
const outDir = fileURLToPath(new URL('../../../backend/desktop/frontend/dist', import.meta.url));

export default defineConfig({
  plugins: [svelte()],
  server: { port: 5175, strictPort: true },
  build: { outDir, emptyOutDir: true },
});
