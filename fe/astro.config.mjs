import { defineConfig } from 'astro/config';
import vue from '@astrojs/vue';
import tailwindcss from '@tailwindcss/vite';
import { loadEnv } from 'vite';

const env = loadEnv(process.env.NODE_ENV || 'development', process.cwd(), '');

const host = env.HOST || 'localhost';
const port = Number(env.PORT) || 4000;
const apiUrl = env.PUBLIC_API_URL || 'http://localhost:8990';
const isProd = (process.env.NODE_ENV || 'development') === 'production';

export default defineConfig({
  // Static output: build is served by nginx (Dockerfile.fe) which also proxies
  // /api to the Go grpc-gateway. No server adapter is required.
  output: 'static',

  // Security defaults for routes rendered on demand. On a fully static site the
  // production boundary (nginx) is the source of truth for headers (see
  // Dockerfile.fe.nginx); `checkOrigin` adds CSRF protection should any route
  // opt out of prerendering in the future.
  security: {
    checkOrigin: true,
  },

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
        // Realtime WebSocket: forwarded to the Go gateway so the frontend can
        // use a same-origin /ws URL in both dev and production.
        '/ws': {
          target: apiUrl,
          changeOrigin: true,
          ws: true,
        },
      },
    },
    build: {
      // Reduce the chance of leaking source maps / volume into production.
      sourcemap: !isProd,
    },
    resolve: {
      alias: {
        '@': '/src',
      },
    },
  },
});
