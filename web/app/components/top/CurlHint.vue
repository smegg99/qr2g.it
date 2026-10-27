<!-- app/components/top/CurlHint.vue -->
<!-- The same shorthand from a terminal; it follows the repository typed above. -->
<template>
  <section
    class="curl"
    :aria-labelledby="headingId"
  >
    <h2
      :id="headingId"
      class="curl__title"
    >
      {{ t('top.terminal') }}
    </h2>
    <div class="curl__line">
      <code class="curl__command"><span
        class="curl__prompt"
        aria-hidden="true"
      >$ </span>curl "{{ base }}<span
        class="curl__option"
        :title="t('top.curlQr')"
      >?qr</span>"</code>
      <span class="curl__note">{{ t('top.curlQr') }}</span>
      <button
        type="button"
        class="curl__copy"
        :class="{ 'curl__copy--done': copied }"
        :aria-label="`${t('top.copy')}: ${command}`"
        :title="t('top.copy')"
        @click="copy(command)"
      >
        <v-icon
          :icon="copied ? 'mdi-check' : 'mdi-content-copy'"
          size="14"
        />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { parseRepoInput } from '~/utils/repository'

const props = defineProps<{ repository: string }>()
const { t } = useI18n()
const headingId = useId()
const { copied, copy } = useCopy()

const base = computed(() => {
  const parsed = parseRepoInput(props.repository)
  return `${window.location.origin}/${parsed ? `${parsed.owner}/${parsed.name}` : 'owner/repo'}`
})
const command = computed(() => `curl "${base.value}?qr"`)
</script>

<style scoped>
.curl { margin-top: 28px; text-align: left; }
.curl__title { margin: 0 0 8px; font-size: 13px; font-weight: 650; color: rgb(var(--v-theme-ink2)); text-align: center; }
.curl__line {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  padding: 2px 4px 2px 12px;
  border: 1px solid rgb(var(--v-theme-line));
  border-radius: var(--pane-radius);
  background: rgb(var(--v-theme-chrome));
}
.v-application .curl__command { padding: 0; border: 0; border-radius: 0; background: none; overflow: hidden; font: 12.5px/28px var(--font-mono); color: rgb(var(--v-theme-ink)); text-overflow: ellipsis; white-space: nowrap; }
.curl__prompt { color: rgb(var(--v-theme-ink3)); user-select: none; }
/* The optional part: without it the command prints only the link. */
.curl__option { font-weight: 700; color: rgb(var(--v-theme-accent)); }
.curl__copy { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; padding: 0; border: 0; border-radius: var(--s99ab-radius); background: none; color: rgb(var(--v-theme-ink3)); cursor: pointer; }
.curl__copy:hover { background: rgb(var(--v-theme-chrome2)); color: rgb(var(--v-theme-ink)); }
.curl__copy--done { color: rgb(var(--v-theme-success)); }
.curl__note { font-size: 11px; color: rgb(var(--v-theme-ink3)); white-space: nowrap; }
@media (max-width: 600px) { .curl__note { display: none; } }
</style>
