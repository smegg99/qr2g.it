<!-- app/components/motion/SlotText.vue -->
<!-- A slot-machine reveal: every character spins through random symbols and lands, left to right. -->
<template>
  <span
    class="slot"
    :aria-label="text"
  >
    <span
      v-for="(reel, i) in reels"
      :key="`${text}-${i}`"
      class="slot__window"
      aria-hidden="true"
    >
      <span
        class="slot__reel"
        :style="{
          transform: `translateY(-${spun ? reel.length - 1 : 0}lh)`,
          transitionDuration: `${duration + i * stagger}ms`,
        }"
      >
        <span
          v-for="(symbol, j) in reel"
          :key="j"
          class="slot__symbol"
        >{{ symbol }}</span>
      </span>
    </span>
  </span>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  text: string
  symbols?: string
  spins?: number
  duration?: number
  stagger?: number
}>(), { symbols: '0123456789ABCDEFGHJKLMNPQRSTUVWXYZ', spins: 9, duration: 650, stagger: 110 })

const spun = ref(false)

// Random symbols above the final one; later reels spin a little longer, like a real machine.
const reels = computed(() => [...props.text].map((final, i) => {
  const count = props.spins + i * 2
  const pick = () => props.symbols[Math.floor(Math.random() * props.symbols.length)]!
  return [...Array.from({ length: count }, pick), final]
}))

function spin() {
  spun.value = false
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
    spun.value = true
    return
  }
  // Two frames: the reels render at the top first, then the transition runs to the final symbol.
  requestAnimationFrame(() => requestAnimationFrame(() => (spun.value = true)))
}

onMounted(spin)
watch(() => props.text, () => nextTick(spin))
</script>

<style scoped>
.slot { display: inline-flex; }
.slot__window { display: inline-block; height: 1lh; overflow: hidden; }
.slot__reel { display: flex; flex-direction: column; transition-property: transform; transition-timing-function: cubic-bezier(0.2, 0.9, 0.25, 1.05); }
.slot__symbol { display: block; height: 1lh; text-align: center; }
@media (prefers-reduced-motion: reduce) { .slot__reel { transition: none; } }
</style>
