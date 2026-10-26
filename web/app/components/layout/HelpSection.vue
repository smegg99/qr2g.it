<!-- app/components/layout/HelpSection.vue -->
<template>
  <section class="help-section">
    <h3>{{ title }}</h3>
    <component
      :is="ordered ? 'ol' : 'ul'"
      :class="['help-list', { 'help-list--ordered': ordered }]"
    >
      <li
        v-for="(item, idx) in items"
        :key="idx"
      >
        {{ item }}
      </li>
    </component>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  items: string[]
  ordered?: boolean
}>(), {
  ordered: false,
})
</script>
