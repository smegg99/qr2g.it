<!-- app/components/generator/LanguageBar.vue -->
<!-- The repository's stack as GitHub shows it: one bar of language shares and a legend. -->
<template>
  <div class="stack">
    <div
      class="stack__bar"
      role="img"
      :aria-label="summary"
    >
      <span
        v-for="part in parts"
        :key="part.name"
        class="stack__part"
        :style="{ flexGrow: part.share, background: part.color }"
      />
    </div>
    <ul class="stack__legend">
      <li
        v-for="part in parts.filter(p => p.name)"
        :key="part.name"
      >
        <i
          class="stack__dot"
          :style="{ background: part.color }"
          aria-hidden="true"
        />{{ part.name }}<span class="stack__share">{{ n(part.share, { style: 'percent', maximumFractionDigits: 1 }) }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import type { LanguageShare } from '~/generated/api/types.gen'
import { languageColor } from '~/composables/useLanguageTheme'

const props = defineProps<{ languages: LanguageShare[] }>()
const { n } = useI18n()

// Languages beyond the six GitHub's largest leave a remainder, drawn as a neutral part without a name.
const parts = computed(() => {
  const named = props.languages.map(l => ({ name: l.name, share: l.share, color: languageColor(l.name) }))
  const rest = 1 - named.reduce((sum, l) => sum + l.share, 0)
  return rest > 0.005 ? [...named, { name: '', share: rest, color: 'rgb(var(--v-theme-line))' }] : named
})
const summary = computed(() => props.languages.map(l => `${l.name} ${n(l.share, { style: 'percent', maximumFractionDigits: 1 })}`).join(', '))
</script>

<style scoped>
.stack { display: flex; flex-direction: column; gap: 8px; width: 100%; }
.stack__bar { display: flex; gap: 2px; height: 8px; overflow: hidden; border-radius: 4px; }
.stack__part { min-width: 2px; }
.stack__legend { display: flex; flex-wrap: wrap; gap: 4px 14px; margin: 0; padding: 0; list-style: none; font-size: 12px; color: rgb(var(--v-theme-ink2)); }
.stack__legend li { display: inline-flex; align-items: center; white-space: nowrap; }
.stack__dot { width: 8px; height: 8px; margin-right: 6px; border-radius: 50%; }
.stack__share { margin-left: 4px; color: rgb(var(--v-theme-ink3)); font-variant-numeric: tabular-nums; }
</style>
