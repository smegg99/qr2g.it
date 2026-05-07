// app/composables/useMaterialAccentTheme.ts
import type { MaybeRefOrGetter } from 'vue'
import {
  computed,
  onBeforeUnmount,
  onMounted,
  readonly,
  ref,
  toValue,
  watch,
} from 'vue'
import { useTheme } from 'vuetify'
import {
  buildMaterialAccentColors,
  getDailyAccentHex,
  normalizeHexColor,
  type MaterialAccentColors,
} from '~/theme/materialAccent'

export function useDailyAccentHex() {
  const accentHex = ref(getDailyAccentHex())
  let midnightTimer: ReturnType<typeof setTimeout> | null = null

  function refresh() {
    accentHex.value = getDailyAccentHex()
  }

  function scheduleNextRefresh() {
    if (midnightTimer) clearTimeout(midnightTimer)

    const now = new Date()
    const nextMidnight = new Date(now)
    nextMidnight.setHours(24, 0, 1, 0)

    midnightTimer = setTimeout(() => {
      refresh()
      scheduleNextRefresh()
    }, nextMidnight.getTime() - now.getTime())
  }

  onMounted(scheduleNextRefresh)

  onBeforeUnmount(() => {
    if (midnightTimer) clearTimeout(midnightTimer)
  })

  return readonly(accentHex)
}

export function useMaterialAccentTheme(sourceHex: MaybeRefOrGetter<string>) {
  const theme = useTheme()
  const source = computed(() => normalizeHexColor(toValue(sourceHex)))
  const isDark = computed(() => theme.global.current.value.dark)
  const colors = ref<MaterialAccentColors>(buildMaterialAccentColors(source.value, isDark.value))

  function applyAccent(hex = source.value) {
    const schemeColors = buildMaterialAccentColors(hex, theme.global.current.value.dark)
    const currentTheme = theme.themes.value[theme.global.name.value]

    colors.value = schemeColors

    if (currentTheme?.colors) {
      Object.assign(currentTheme.colors, schemeColors)
    }
  }

  watch(
    [source, () => theme.global.name.value, isDark],
    () => applyAccent(),
    { immediate: true },
  )

  return {
    colors: readonly(colors),
    source,
    applyAccent,
  }
}
