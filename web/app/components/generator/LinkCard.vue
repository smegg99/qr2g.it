<!-- app/components/generator/LinkCard.vue -->
<template>
  <div :class="['display', { 'display--copied': copied }]">
    <div class="display__address">
      <span class="display__label">{{ t('result.shortLink') }}</span>
      <button
        type="button"
        class="display__slug"
        :aria-label="`${t('result.copy')}: ${link.url}`"
        :title="t('result.copy')"
        @click="copy(link.url)"
      >
        <span class="display__host">{{ parts.host }}/</span>
        <SlotText
          class="display__code"
          :text="parts.slug"
        />
      </button>
    </div>
    <div class="display__keys">
      <v-btn
        :color="copied ? 'success' : 'accent'"
        :prepend-icon="copied ? 'mdi-check' : 'mdi-content-copy'"
        variant="flat"
        class="display__copy"
        @click="copy(link.url)"
      >
        {{ copied ? t('result.copied') : t('result.copy') }}
      </v-btn>
      <v-btn
        :href="link.url"
        target="_blank"
        rel="noopener"
        icon="mdi-open-in-new"
        variant="outlined"
        :aria-label="t('result.openShort')"
        :title="t('result.openShort')"
      />
    </div>
    <LinkStats
      class="display__stats"
      :link="link"
    />
    <div class="display__footer">
      <template v-if="link.currentFullName">
        <span class="display__label">{{ t('result.repository') }}</span>
        <a
          :href="link.currentUrl"
          target="_blank"
          rel="noopener"
          class="display__repo"
          :aria-label="`${t('result.openRepo')}: ${link.currentFullName}`"
        >
          <v-icon
            icon="mdi-github"
            size="16"
            aria-hidden="true"
          />
          <span>{{ link.currentFullName }}</span>
        </a>
      </template>
      <span
        v-if="outcome === 'created'"
        class="display__outcome"
      >{{ t('generator.outcome.created') }}</span>
      <LanguageBar
        v-if="link.languages?.length"
        :languages="link.languages"
      />
    </div>
    <span
      class="sr-only"
      role="status"
    >{{ copied ? t('result.copied') : '' }}</span>
  </div>
</template>

<script setup lang="ts">
import type { Link } from '~/generated/api/types.gen'
import type { Outcome } from '~/composables/useGenerator'
import SlotText from '~/components/motion/SlotText.vue'
import LanguageBar from './LanguageBar.vue'
import LinkStats from './LinkStats.vue'

const props = defineProps<{ link: Link, outcome: Outcome | null }>()
const { t } = useI18n()
const { copied, copy } = useCopy()

const parts = computed(() => {
  const bare = props.link.url.replace(/^https?:\/\//i, '').replace(/\/$/, '')
  const cut = bare.lastIndexOf('/')
  return cut < 0 ? { host: bare, slug: props.link.slug } : { host: bare.slice(0, cut), slug: bare.slice(cut + 1) }
})
</script>

<style scoped>
.display {
  --display-pad: 24px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px 24px;
  padding: var(--display-pad) var(--display-pad) 0;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 80%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, rgb(var(--v-theme-well)) 70%, transparent);
  transition: border-color 0.2s ease;
}
.display__stats { grid-column: 1 / -1; align-self: stretch; height: 100%; }
.display__address { display: flex; flex-direction: column; align-items: flex-start; gap: 6px; min-width: 0; }
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
  padding: 12px var(--display-pad);
  border-top: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 60%, transparent);
  background: color-mix(in srgb, rgb(var(--v-theme-chrome2)) 35%, transparent);
}
.display__repo { display: inline-flex; align-items: center; gap: 6px; min-width: 0; max-width: 100%; font-size: 14px; font-weight: 600; color: rgb(var(--v-theme-ink)); text-decoration: none; }
.display__repo span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.display__repo:hover { color: rgb(var(--v-theme-accent)); }
.display__footer a:focus-visible { outline-offset: -2px; }
.display__outcome { margin-left: auto; font-size: 12px; color: rgb(var(--v-theme-ink3)); }

@media (max-width: 600px) {
  .display { --display-pad: 18px; grid-template-columns: minmax(0, 1fr); gap: 16px; }
  .display__copy { flex: 1 1 auto; }
  .display__outcome { margin-left: 0; }
}
</style>
