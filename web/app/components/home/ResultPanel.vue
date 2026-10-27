<!-- app/components/home/ResultPanel.vue -->
<template>
  <div class="pane-body">
    <div :class="['display', { 'display--copied': copied }]">
      <div class="display__address">
        <span class="display__label">{{ t('home.result.shortLink') }}</span>
        <button
          type="button"
          class="display__slug"
          :aria-label="`${t('home.result.copy')}: ${link.url}`"
          :title="t('home.result.copy')"
          @click="copy"
        >
          <span class="display__host">{{ parts.host }}/</span>
          <span class="display__code">{{ parts.slug }}</span>
        </button>
      </div>
      <div class="display__keys">
        <v-btn
          :color="copied ? 'success' : 'accent'"
          :prepend-icon="copied ? 'mdi-check' : 'mdi-content-copy'"
          variant="flat"
          class="display__copy"
          @click="copy"
        >
          {{ copied ? t('home.result.copied') : t('home.result.copy') }}
        </v-btn>
        <v-btn
          :href="link.url"
          target="_blank"
          rel="noopener"
          icon="mdi-open-in-new"
          variant="outlined"
          :aria-label="t('home.result.openShort')"
          :title="t('home.result.openShort')"
        />
      </div>
      <div class="display__footer">
        <span class="display__label">{{ t('home.result.repository') }}</span>
        <a
          :href="link.currentUrl"
          target="_blank"
          rel="noopener"
          class="display__repo"
          :aria-label="`${t('home.result.openRepo')}: ${repoName}`"
        >
          <v-icon
            icon="mdi-github"
            size="16"
            aria-hidden="true"
          />
          <span>{{ repoName }}</span>
        </a>
      </div>
      <span
        class="sr-only"
        role="status"
      >{{ copied ? t('home.result.copied') : '' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { LinkResponse } from '~/types/api'

const props = defineProps<{
  link: LinkResponse
}>()

const { t } = useI18n()
const copied = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

const repoName = computed(() => props.link.currentFullName || props.link.initialFullName)

const parts = computed(() => {
  const bare = props.link.url.replace(/^https?:\/\//i, '').replace(/\/$/, '')
  const cut = bare.indexOf('/')
  return cut < 0 ? { host: bare, slug: props.link.slug } : { host: bare.slice(0, cut), slug: bare.slice(cut + 1) }
})

async function copy() {
  try {
    await navigator.clipboard.writeText(props.link.url)
    copied.value = true
    clearTimeout(timer)
    timer = setTimeout(() => (copied.value = false), 1600)
  }
  catch {
    // Clipboard access is optional in locked-down browser contexts.
  }
}

onBeforeUnmount(() => clearTimeout(timer))
</script>

<style scoped>
/* One address readout, with a separate repository footer. */
.display {
  --display-pad: 24px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 24px;
  padding: var(--display-pad) var(--display-pad) 0;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 80%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, rgb(var(--v-theme-well)) 70%, transparent);
  transition: border-color 0.2s ease;
}
.display__address { display: flex; flex-direction: column; align-items: flex-start; gap: 8px; min-width: 0; }
.display__label { font-size: 12px; font-weight: 600; color: rgb(var(--v-theme-ink3)); }
.display__slug {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  max-width: 100%;
  padding: 0;
  border: 0;
  border-radius: var(--s99ab-radius);
  background: transparent;
  color: rgb(var(--v-theme-ink));
  font: 600 clamp(24px, 3.2vw, 36px)/1.25 var(--font-mono);
  letter-spacing: -0.04em;
  text-align: left;
  overflow-wrap: anywhere;
  cursor: copy;
}
.display__host { min-width: 0; color: rgb(var(--v-theme-ink2)); }
.display__code { min-width: 0; font-weight: 800; transition: color 0.2s ease; }
.display__slug:hover .display__code { color: rgb(var(--v-theme-accent)); }
.display--copied { border-color: rgb(var(--v-theme-success)); }
.display--copied .display__code { color: rgb(var(--v-theme-success)); }
.display__keys { --s99ab-control-h: 40px; display: flex; align-items: center; gap: 8px; }
.display__copy { min-width: 128px; font-weight: 700; text-transform: none; letter-spacing: 0; }
.display__footer {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  min-width: 0;
  margin-inline: calc(-1 * var(--display-pad));
  padding: 14px var(--display-pad);
  border-top: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 60%, transparent);
  background: color-mix(in srgb, rgb(var(--v-theme-chrome2)) 35%, transparent);
}
.display__repo { display: inline-flex; align-items: center; gap: 6px; min-width: 0; max-width: 100%; font-size: 14px; font-weight: 600; color: rgb(var(--v-theme-ink)); text-decoration: none; }
.display__repo span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.display__repo:hover { color: rgb(var(--v-theme-accent)); }
.display__footer a:focus-visible { outline-offset: -2px; }

@media (max-width: 600px) {
  .display { --display-pad: 18px; grid-template-columns: minmax(0, 1fr); gap: 18px; }
  .display__copy { flex: 1 1 auto; }
}
</style>
