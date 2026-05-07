<!-- app/components/home/QrControlField.vue -->
<template>
  <div class="d-flex flex-column ga-2">
    <div class="text-caption text-medium-emphasis">
      {{ label }}
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{ label: string }>()
</script>
