import { sveltekit } from '@sveltejs/kit/vite';
import adapter from '@sveltejs/adapter-static';
import { defineConfig } from 'vite';

const api = process.env.VITE_DEV_API ?? 'http://localhost:8080';

export default defineConfig({
  // Static SPA build: served by nginx in production, API on /api.
  plugins: [sveltekit({ adapter: adapter({ fallback: 'index.html' }) })],
  server: {
    port: 5174,
    strictPort: true,
    // Same-origin in dev: the browser calls /api, Vite forwards to Go.
    proxy: {
      '/api': api,
      '/healthz': api,
      '/readyz': api,
    },
  },
});
