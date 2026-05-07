<!-- app/components/common/GlassCard.vue -->
<template>
  <v-card :class="['glass-card', { 'glass-card--strong': strong }]" :rounded="rounded" :border="border" v-bind="$attrs">
    <slot />
  </v-card>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  strong?: boolean
  rounded?: string | number | boolean
  border?: boolean | string
}>(), {
  strong: false,
  rounded: 'lg',
  border: 'thin opacity-50',
})
</script>

<style scoped>
.glass-card {
  background: rgba(var(--v-theme-surface), 0.55) !important;
  backdrop-filter: blur(18px) saturate(140%);
  -webkit-backdrop-filter: blur(18px) saturate(140%);
  isolation: isolate;
  transform: translateZ(0);
  will-change: transform;
}

.glass-card--strong {
  background: rgba(var(--v-theme-surface), 0.78) !important;
  backdrop-filter: blur(22px) saturate(160%);
  -webkit-backdrop-filter: blur(22px) saturate(160%);
}
</style>
