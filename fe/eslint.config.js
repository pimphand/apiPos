// ESLint 9 flat config.
// TypeScript + JavaScript sources only. Vue (.vue) and Astro (.astro) files are
// type-checked via `vue-tsc` and Astro's own tooling, and are intentionally
// excluded here (their parsers/plugins aren't part of the installed toolchain).
import js from '@eslint/js';
import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';

export default [
  {
    ignores: [
      'dist/**',
      '.astro/**',
      'node_modules/**',
      'coverage/**',
      'test-results/**',
      // Vue/Astro are type-checked by vue-tsc + Astro tooling (no separate
      // parsers installed in this JS-only lint stack).
      '**/*.vue',
      '**/*.astro',
    ],
  },
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx,js,jsx,mjs,cjs}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
    plugins: {
      '@typescript-eslint': tsPlugin,
    },
    rules: {
      ...tsPlugin.configs.recommended.rules,
      // TypeScript's type checker reports undefined variables; no-undef is
      // noisy for browser globals (window/document) and Node globals (process).
      'no-undef': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      'no-console': 'warn',
    },
  },
];
