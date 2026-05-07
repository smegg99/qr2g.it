<!-- app/components/home/ResultRepoTile.vue -->
<template>
  <GlassCard class="repo-tile pa-4 pa-sm-5" :href="href" target="_blank" rel="noopener" elevation="0" :ripple="false">
    <div class="d-flex align-center ga-4">
      <v-icon icon="mdi-github" class="repo-tile__icon" />

      <div class="repo-tile__main">
        <div class="text-caption text-medium-emphasis">
          {{ repoLabel }}
        </div>
        <div class="text-body-1 font-weight-medium repo-tile__name">
          {{ title }}
        </div>
      </div>

      <v-divider vertical :opacity="0.5" class="mx-2 repo-tile__divider" />

      <div class="repo-tile__status">
        <div class="text-caption text-medium-emphasis">
          {{ statusLabel }}
        </div>
        <div :class="['font-weight-bold repo-tile__status-text', statusColorClass]">
          {{ statusText }}
        </div>
      </div>

      <v-icon icon="mdi-open-in-new" size="16" class="repo-tile__open" />
    </div>
  </GlassCard>
</template>

<script setup lang="ts">
import GlassCard from '~/components/common/GlassCard.vue'

type StateKey = 'active' | 'tombstoned' | 'stale' | 'unknown'

const props = defineProps<{
  href: string
  title: string
  repoLabel: string
  statusLabel: string
  statusText: string
  state: StateKey
}>()

const statusColorClass = computed(() => {
  switch (props.state) {
    case 'active': return 'text-success'
    case 'stale': return 'text-warning'
    case 'tombstoned': return 'text-error'
    default: return 'text-medium-emphasis'
  }
})
</script>

<style scoped>
.repo-tile {
  background: rgba(var(--v-theme-surface), 0.34) !important;
  border-color: rgba(var(--v-theme-on-surface), 0.1) !important;
  text-decoration: none;
  box-shadow: none !important;
  transition: background 0.18s ease, border-color 0.18s ease;
}

.repo-tile:hover {
  background: rgba(var(--v-theme-surface), 0.46) !important;
  border-color: rgba(var(--v-theme-primary), 0.28) !important;
  box-shadow: none !important;
}

.repo-tile__icon {
  font-size: 40px !important;
  flex: 0 0 40px;
  opacity: 0.92;
}

.repo-tile__main {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.repo-tile__name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.repo-tile__divider {
  height: 36px;
  flex: 0 0 auto;
}

.repo-tile__status {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.repo-tile__status-text {
  font-size: 13px;
}

.repo-tile__open {
  flex: 0 0 auto;
  opacity: 0.5;
}

@media (max-width: 600px) {
  .repo-tile__icon {
    font-size: 30px !important;
    flex-basis: 30px;
  }
}
</style>
