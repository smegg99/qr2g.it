<!-- app/components/home/ErrorMessage.vue -->
<template>
  <v-slide-y-transition
    appear
    mode="out-in"
    :duration="{ enter: 240, leave: 160 }"
  >
    <v-alert
      v-if="code"
      :key="alertKey"
      color="error"
      icon="mdi-alert-circle-outline"
      variant="tonal"
      density="comfortable"
      closable
      role="alert"
      @click:close="emit('dismiss')"
    >
      <div class="text-body-2 font-weight-medium">
        {{ headline }}
      </div>
    </v-alert>
  </v-slide-y-transition>
</template>

<script setup lang="ts">
import type { ApiErrorCode } from '~/types/api'

const props = defineProps<{
  code: ApiErrorCode | null
}>()

const emit = defineEmits<{
  (e: 'dismiss'): void
}>()

const { t, te } = useI18n()

const headline = computed(() => {
  if (!props.code) return ''
  const key = `home.errors.${props.code}`
  if (te(key)) return t(key)
  return t('home.errors.unknown')
})

const alertKey = computed(() => props.code ?? 'none')
</script>
