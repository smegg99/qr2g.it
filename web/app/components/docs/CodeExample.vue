<!-- app/components/docs/CodeExample.vue -->
<template>
  <div class="code-example">
    <pre><code>{{ code }}</code></pre>
    <v-btn
      class="code-copy"
      :icon="copied ? 'mdi-check' : 'mdi-content-copy'"
      variant="text"
      size="small"
      :aria-label="t(copied ? 'docs.copied' : 'docs.copy')"
      :title="t(copied ? 'docs.copied' : 'docs.copy')"
      @click="copy"
    />
    <span
      class="sr-only"
      role="status"
    >{{ message }}</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ code: string }>()
const { t } = useI18n()
const copied = ref(false)
const message = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
async function copy() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    message.value = t('docs.copied')
    clearTimeout(timer)
    timer = setTimeout(() => {
      copied.value = false
      message.value = ''
    }, 1800)
  }
  catch { message.value = t('docs.copyFailed') }
}
onBeforeUnmount(() => clearTimeout(timer))
</script>
