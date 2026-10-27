<!-- app/components/motion/Reveal.vue -->
<!-- Opens and closes its content smoothly: the height and the opacity animate together, nothing jumps. -->
<template>
  <Transition name="reveal">
    <div
      v-if="show"
      class="reveal"
    >
      <div class="reveal__inner">
        <div :style="gap ? { paddingTop: `${gap}px` } : undefined">
          <slot />
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
// `gap` is space above the content that opens and closes with it, instead of a margin that would jump.
defineProps<{ show: boolean, gap?: number }>()
</script>

<style scoped>
.reveal { display: grid; grid-template-rows: 1fr; }
.reveal__inner { min-height: 0; overflow: hidden; }
.reveal-enter-active, .reveal-leave-active { transition: grid-template-rows 0.32s cubic-bezier(0.2, 0.8, 0.2, 1), opacity 0.26s ease; }
.reveal-enter-from, .reveal-leave-to { grid-template-rows: 0fr; opacity: 0; }
@media (prefers-reduced-motion: reduce) { .reveal-enter-active, .reveal-leave-active { transition: opacity 0.15s ease; } }
</style>
