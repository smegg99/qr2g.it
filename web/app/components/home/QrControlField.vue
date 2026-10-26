<!-- app/components/home/QrControlField.vue -->
<template>
  <div class="control">
    <div class="control__label">
      <span>{{ label }}</span>
      <span
        v-if="hint"
        class="control__hint"
      >{{ hint }}</span>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{ label: string, hint?: string }>()
</script>
