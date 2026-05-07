// nuxt.config.ts
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import svgLoader from 'vite-svg-loader'

// Pull repo-root .env (one level above /web) into process.env so the same
// TURNSTILE_SITE_KEY / QR2G_BACKEND_URL the Go backend reads is visible here.
// /web/.env (if any) still wins because Nuxt loads it after this file is
// evaluated; we only set keys that are not already defined.
const here = path.dirname(fileURLToPath(import.meta.url))
const rootEnvPath = path.join(here, '..', '.env')
if (fs.existsSync(rootEnvPath)) {
  for (const raw of fs.readFileSync(rootEnvPath, 'utf8').split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const eq = line.indexOf('=')
    if (eq < 1) continue
    const key = line.slice(0, eq).trim()
    if (!/^[A-Z_][A-Z0-9_]*$/.test(key)) continue
    if (process.env[key] !== undefined) continue
    let value = line.slice(eq + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith('\'') && value.endsWith('\''))) {
      value = value.slice(1, -1)
    }
    process.env[key] = value
  }
}

// Cloudflare's documented "always passes" test site key. Used as a dev
// fallback so a fresh checkout works without setting any env var; override
// for production via NUXT_PUBLIC_TURNSTILE_SITE_KEY or TURNSTILE_SITE_KEY.
const TURNSTILE_TEST_SITE_KEY = '1x00000000000000000000AA'

const turnstileSiteKey
  = process.env.NUXT_PUBLIC_TURNSTILE_SITE_KEY
    || process.env.TURNSTILE_SITE_KEY
    || TURNSTILE_TEST_SITE_KEY

const localeFiles = ['common.json']

export default defineNuxtConfig({
  modules: [
    '@nuxtjs/i18n',
    '@pinia/nuxt',
    'pinia-plugin-persistedstate/nuxt',
    '@nuxt/eslint',
  ],
  ssr: false,
  devtools: { enabled: true },
  app: {
    pageTransition: { name: 'page', mode: 'out-in' },
    head: {
      title: 'qr2g.it',
      meta: [
        { name: 'description', content: 'Permanent, QR-friendly short links for public GitHub repositories.' },
        { name: 'theme-color', content: '#08090C' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
      ],
    },
  },
  css: [
    '~/assets/css/fonts.css',
    '~/assets/css/transitions.css',
    '~/assets/css/vuetify.scss',
  ],
  runtimeConfig: {
    public: {
      apiBase: '/api/v1',
      shortBase: 'https://qr2g.it',
      turnstileSiteKey,
    },
  },
  build: {
    transpile: ['vuetify', '@material/material-color-utilities'],
  },
  devServer: { port: 5173 },
  features: {
    inlineStyles: false,
  },
  compatibilityDate: '2025-07-15',
  nitro: {
    compressPublicAssets: true,
  },
  vite: {
    optimizeDeps: {
      include: ['vuetify', 'qrcode', '@firecms/neat'],
    },
    plugins: [
      vuetify({
        autoImport: true,
        styles: { configFile: 'assets/css/vuetify.scss' },
      }),
      svgLoader({
        defaultImport: 'component',
      }),
    ],
    server: {
      proxy: {
        '/api': {
          target: process.env.QR2G_BACKEND_URL || 'http://127.0.0.1:9245',
          changeOrigin: true,
        },
      },
    },
    vue: {
      template: {
        transformAssetUrls,
      },
    },
  },
  eslint: {
    config: {
      stylistic: true,
    },
  },
  i18n: {
    strategy: 'no_prefix',
    defaultLocale: 'en',
    langDir: 'locales',
    locales: [
      { code: 'en', name: 'English', language: 'en-GB', files: localeFiles.map(f => `en/${f}`) },
      { code: 'pl', name: 'Polski', language: 'pl-PL', files: localeFiles.map(f => `pl/${f}`) },
    ],
    compilation: {
      strictMessage: false,
      escapeHtml: false,
    },
    detectBrowserLanguage: false,
    vueI18n: './i18n/i18n.config.ts',
  },
})
