import { defineConfig } from 'astro/config';
import vue from '@astrojs/vue';
import node from '@astrojs/node';
import tailwindcss from '@tailwindcss/vite';
import { loadEnv } from 'vite';

const env = loadEnv(process.env.NODE_ENV || 'development', process.cwd(), '');

const host = env.HOST || 'localhost';
const port = Number(env.PORT) || 4000;
const apiUrl = env.PUBLIC_API_URL || 'http://localhost:8990';

export default defineConfig({
  output: 'static',
  adapter: node({
    mode: 'standalone',
  }),
  integrations: [
    vue({
      appEntrypoint: '/src/pages/_app.ts',
    }),
  ],
  server: {
    host,
    port,
  },
  vite: {
    plugins: [tailwindcss()],
    server: {
      // Astro v5 no longer reads `server.proxy`; the dev proxy must live in the
      // Vite config. Forwards /api/* to the POS backend HTTP gateway.
      proxy: {
        '/api': {
          target: apiUrl,
          changeOrigin: true,
        },
      },
    },
    resolve: {
      alias: {
        '@': '/src',
      },
    },
  },
});