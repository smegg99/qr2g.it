<!-- app/components/generator/ErrorMessage.vue -->
<template>
  <div
    class="notice"
    role="alert"
  >
    <v-icon
      icon="mdi-alert-circle-outline"
      size="20"
      class="notice__icon"
      aria-hidden="true"
    />
    <span class="notice__text">{{ text }}</span>
    <span
      v-if="retry"
      class="notice__actions"
    >
      <v-btn
        v-if="retry"
        variant="text"
        size="small"
        class="notice__retry"
        :text="t('generator.retry')"
        @click="emit('retry')"
      />
    </span>
  </div>
</template>

<script setup lang="ts">
import type { ApiError } from '~/utils/api'

const props = defineProps<{ error: ApiError, retry?: boolean }>()
const emit = defineEmits<{ (e: 'retry'): void }>()
const { t } = useI18n()
const errorText = useErrorText()
const text = computed(() => errorText(props.error))
</script>

<style scoped>
/* Same radius and border weight as the address card, tinted with the error colour. */
.notice {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 6px 6px 6px 14px;
  border: 1px solid color-mix(in srgb, rgb(var(--v-theme-error)) 45%, transparent);
  border-radius: var(--notice-radius, 8px);
  background: color-mix(in srgb, rgb(var(--v-theme-error)) 10%, rgb(var(--v-theme-chrome)));
  color: rgb(var(--v-theme-ink));
  text-align: left;
}
.notice__icon { flex: 0 0 auto; color: rgb(var(--v-theme-error)); }
.notice__text { flex: 1 1 auto; min-width: 0; font-size: 14px; font-weight: 500; line-height: 1.4; }
.notice__actions { display: flex; align-items: center; gap: 2px; flex: 0 0 auto; --s99ab-control-h: 32px; }
.notice__retry { text-transform: none; letter-spacing: 0; font-weight: 600; color: rgb(var(--v-theme-error)); }
</style>
