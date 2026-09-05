import { defineConfig } from '@playwright/test';

export default defineConfig({
  projects: [
    {
      name: 'chrome',
      use: {
        channel: 'chrome',
        headless: false,
      },
    },
  ],
});
