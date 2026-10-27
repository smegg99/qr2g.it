// web/nuxt.config.ts
import { readFileSync, readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { palette } from '@smegg99/s99ab/theme'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import svgLoader from 'vite-svg-loader'
import { BRAND } from './app/theme/brand'
import { ACCENT_KEY, PAINT_KEY } from './app/theme/visitor'

const version = readFileSync(fileURLToPath(new URL('../VERSION', import.meta.url)), 'utf8').trim()
if (!/^\d+\.\d+\.\d+$/.test(version)) throw new Error('VERSION must contain MAJOR.MINOR.PATCH')
const localeFiles = readdirSync(fileURLToPath(new URL('./i18n/generated/en', import.meta.url)))
  .filter(file => file.endsWith('.json')).sort()
const isDev = process.argv.includes('dev')
const siteKey = process.env.NUXT_PUBLIC_TURNSTILE_SITE_KEY || process.env.TURNSTILE_SITE_KEY || (isDev ? '1x00000000000000000000AA' : '')
const grounds = {
  dark: palette({ name: BRAND.palette, dark: true, chroma: BRAND.chroma }).bg,
  light: palette({ name: BRAND.palette, dark: false, chroma: BRAND.chroma }).bg,
}

export default defineNuxtConfig({
  extends: ['@smegg99/s99ab'],
  modules: ['@nuxtjs/i18n', '@pinia/nuxt', 'pinia-plugin-persistedstate/nuxt', '@nuxt/eslint'],
  ssr: false,
  devtools: { enabled: false },
  app: {
    pageTransition: false,
    head: {
      title: 'qr2g.it',
      meta: [
        { name: 'description', content: 'Permanent, QR-friendly short links for public GitHub repositories.' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
      ],
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
      script: [{
        innerHTML: `(function(){try{var c=document.cookie.split('; ').find(function(x){return x.startsWith('qr2g.settings=')}),s=c?JSON.parse(decodeURIComponent(c.slice('qr2g.settings='.length))):{},m=s.themeMode||'auto',d=m==='dark'||(m==='auto'&&matchMedia('(prefers-color-scheme: dark)').matches),g=${JSON.stringify(grounds)};try{var p=JSON.parse(localStorage.getItem(${JSON.stringify(PAINT_KEY)})||'null');if(p&&p.seed===localStorage.getItem(${JSON.stringify(ACCENT_KEY)})&&/^#[0-9a-f]{6}$/i.test(p.light)&&/^#[0-9a-f]{6}$/i.test(p.dark))g=p}catch(e){}document.documentElement.style.backgroundColor=d?g.dark:g.light;document.documentElement.style.colorScheme=d?'dark':'light'}catch(e){}})()`,
        tagPosition: 'head',
      }],
    },
  },
  css: ['~/assets/css/site.css'],
  runtimeConfig: { public: { apiBase: '/api/v1', shortBase: 'https://qr2g.it', turnstileSiteKey: siteKey, version } },
  build: { transpile: ['vuetify', '@smegg99/s99ab'] },
  devServer: { host: '127.0.0.1', port: 5173 },
  features: { inlineStyles: false },
  compatibilityDate: '2026-10-26',
  nitro: { compressPublicAssets: true, prerender: { routes: ['/docs'] } },
  vite: {
    plugins: [
      vuetify({ autoImport: true, styles: { configFile: 'assets/css/vuetify.scss' } }),
      svgLoader({ defaultImport: 'component' }),
    ],
    server: { proxy: { '/api': { target: process.env.QR2G_BACKEND_URL || 'http://127.0.0.1:9245', changeOrigin: true } } },
    vue: { template: { transformAssetUrls } },
  },
  // Preserve the shared theme's nested calc() values; cssnano's calc parser rejects them.
  postcss: { plugins: { cssnano: { preset: ['default', { calc: false }] } } },
  eslint: { config: { stylistic: true } },
  i18n: {
    strategy: 'no_prefix',
    defaultLocale: 'en',
    langDir: 'generated',
    locales: [
      { code: 'en', name: 'English', language: 'en-GB', files: localeFiles.map(file => `en/${file}`) },
      { code: 'pl', name: 'Polski', language: 'pl-PL', files: localeFiles.map(file => `pl/${file}`) },
    ],
    compilation: { strictMessage: false, escapeHtml: false },
    detectBrowserLanguage: false,
    vueI18n: 'i18n.config.ts',
  },
})
