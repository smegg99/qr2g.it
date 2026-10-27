<!-- app/components/common/PaneHead.vue -->
<template>
  <header class="pane-head">
    <component
      :is="tag"
      :id="id"
      class="pane-title"
    >
      {{ title }}
    </component>
    <span class="pane-meta">
      <slot name="meta" />
    </span>
    <span
      v-if="$slots.actions"
      class="pane-actions"
    >
      <slot name="actions" />
    </span>
  </header>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ title: string, tag?: 'h1' | 'h2' | 'h3', id?: string }>(), { tag: 'h2', id: undefined })
</script>
