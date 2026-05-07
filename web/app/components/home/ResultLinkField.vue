<!-- app/components/home/ResultLinkField.vue -->
<template>
  <div class="d-flex align-stretch ga-2 w-100">
    <v-text-field class="flex-1-1-auto" :model-value="url" :label="label" prepend-inner-icon="mdi-link-variant"
      density="comfortable" hide-details readonly />

    <v-btn :color="copied ? 'success' : 'primary'" :icon="copied ? 'mdi-check' : 'mdi-content-copy'"
      :aria-label="activeCopyLabel" variant="outlined" class="link-action" @click.stop="onCopy" />

    <v-btn :aria-label="openLabel" :href="url" target="_blank" rel="noopener" icon="mdi-open-in-new" color="primary"
      variant="outlined" class="link-action" />
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  url: string
  label: string
  copyLabel: string
  copiedLabel: string
  openLabel: string
}>()

const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null
const activeCopyLabel = computed(() => copied.value ? props.copiedLabel : props.copyLabel)

async function onCopy() {
  try {
    await navigator.clipboard.writeText(props.url)
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => (copied.value = false), 1600)
  }
  catch {
    // Clipboard access is optional in locked-down browser contexts.
  }
}

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>

<style scoped>
.link-action {
  height: 48px;
  width: 48px;
  min-width: 48px;
  flex: 0 0 48px;
  background: rgba(var(--v-theme-surface), 0.55) !important;
  backdrop-filter: blur(14px) saturate(140%);
  -webkit-backdrop-filter: blur(14px) saturate(140%);
  border-color: rgba(var(--v-theme-primary), 0.42) !important;
}

.link-action:hover {
  background: rgba(var(--v-theme-primary), 0.1) !important;
  border-color: rgba(var(--v-theme-primary), 0.72) !important;
}
</style>
