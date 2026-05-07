<!-- app/components/layout/HelpSection.vue -->
<template>
  <div class="d-flex flex-column ga-2">
    <div :class="['text-caption text-uppercase letter-spaced font-weight-medium', titleColor]">
      {{ title }}
    </div>
    <div class="d-flex flex-column ga-2">
      <div v-for="(item, idx) in items" :key="idx" class="d-flex align-baseline ga-2 text-body-2">
        <div :class="['help-marker text-no-wrap font-weight-bold', titleColor]">
          {{ ordered ? `${idx + 1}.` : '·' }}
        </div>
        <div class="help-text">
          {{ item }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  items: string[]
  ordered?: boolean
  titleColor?: string
}>(), {
  ordered: false,
  titleColor: 'text-primary',
})
</script>

<style scoped>
.letter-spaced {
  letter-spacing: 0.08em;
}

.help-marker {
  flex: 0 0 auto;
  min-width: 14px;
  font-variant-numeric: tabular-nums;
}

.help-text {
  flex: 1 1 auto;
  line-height: 1.55;
}
</style>
