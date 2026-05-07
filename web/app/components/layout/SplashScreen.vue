<!-- app/components/layout/SplashScreen.vue -->
<template>
  <v-fade-transition>
    <v-sheet v-if="visible" class="splash" rounded="0" color="#08090C" />
  </v-fade-transition>
</template>

<script setup lang="ts">
const visible = ref(true)

onMounted(() => {
  requestAnimationFrame(() => {
    setTimeout(() => {
      visible.value = false
    }, 50)
  })
})
</script>

<style scoped>
.splash {
  position: fixed;
  inset: 0;
  z-index: 9999;
  pointer-events: none;
  border-radius: 0 !important;
}
</style>
