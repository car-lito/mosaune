import { defineConfig } from 'vite';
import './scripts/build-theme.mjs';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  root: 'web',
  plugins: [svelte()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: { host: '127.0.0.1', proxy: { '/api': 'http://127.0.0.1:47831' } },
});
