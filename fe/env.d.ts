/// <reference types="astro/client" />

interface ImportMetaEnv {
  readonly PUBLIC_API_BASE_URL: string;
  readonly PUBLIC_API_URL: string;
  readonly PUBLIC_APP_NAME: string;
  readonly PUBLIC_SESSION_MAX_AGE: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}