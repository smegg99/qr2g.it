<!-- app/components/top/TopRepositories.vue -->
<!-- The most visited listed repositories; the server applies every listing and opt-out rule. -->
<template>
  <section
    v-if="items.length"
    class="top"
    :aria-labelledby="headingId"
  >
    <h2
      :id="headingId"
      class="top__title"
    >
      {{ t('top.title') }}
    </h2>
    <ul class="top__list">
      <li
        v-for="item in items"
        :key="item.slug"
        class="top__card"
      >
        <a
          :href="item.url"
          target="_blank"
          rel="noopener"
          class="top__name"
          :title="item.description || item.fullName"
        >
          <span class="top__owner">{{ owner(item.fullName) }}/</span><strong>{{ name(item.fullName) }}</strong>
        </a>
        <span class="top__address">{{ item.url.replace(/^https?:\/\//, '') }}</span>
        <Sparkline
          :days="item.days"
          :label="t('top.chartLabel', { name: item.fullName })"
        />
        <dl class="top__facts">
          <div>
            <dt>{{ t('top.total') }}</dt>
            <dd>
              <NumberFlow
                :value="ready ? item.total : 0"
                :locales="locale"
                :spin-timing="SPIN"
              />
            </dd>
          </div>
          <div>
            <dt>{{ t('top.month') }}</dt>
            <dd>
              <NumberFlow
                :value="ready ? item.total30 : 0"
                :locales="locale"
                :spin-timing="SPIN"
              />
            </dd>
          </div>
          <div>
            <dt>{{ t('top.last') }}</dt>
            <dd :title="item.lastRedirectAt ? d(new Date(item.lastRedirectAt), 'medium') : undefined">
              {{ item.lastRedirectAt ? timeAgo(item.lastRedirectAt, locale, t('top.now')) : t('top.never') }}
            </dd>
          </div>
        </dl>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@pinia/colada'
import { listTopRepositoriesQuery } from '~/generated/api/@pinia/colada.gen'
import NumberFlow from '@number-flow/vue'
import { timeAgo } from '~/utils/time'
import Sparkline from './Sparkline.vue'

const { t, d, locale } = useI18n()
const headingId = useId()
const query = useQuery(() => listTopRepositoriesQuery({ query: { limit: 3 } }))
const items = computed(() => query.data.value?.items ?? [])

const owner = (full: string) => full.split('/')[0]
const name = (full: string) => full.split('/')[1]

// The counters roll up from zero once the cards have data, like the result card's.
const SPIN = { duration: 1400, easing: 'cubic-bezier(0.2, 0.9, 0.25, 1.1)' }
const ready = ref(false)
watch(items, list => list.length && setTimeout(() => (ready.value = true), 150), { immediate: true })
</script>

<style scoped>
.top { margin-top: 40px; text-align: left; }
.top__title { margin: 0 0 10px; font-size: 13px; font-weight: 650; color: rgb(var(--v-theme-ink2)); text-align: center; }
.top__list { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin: 0; padding: 0; list-style: none; }
.top__card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: 14px 14px 12px;
  border: 1px solid rgb(var(--v-theme-line));
  border-radius: var(--pane-radius);
  background: rgb(var(--v-theme-chrome));
}
.top__name { overflow: hidden; font-size: 14px; color: rgb(var(--v-theme-ink)); text-decoration: none; text-overflow: ellipsis; white-space: nowrap; }
.top__name:hover strong { color: rgb(var(--v-theme-accent)); }
.top__owner { color: rgb(var(--v-theme-ink3)); }
.top__address { margin-top: -6px; font: 12px/1.3 var(--font-mono); color: rgb(var(--v-theme-ink3)); }
.top__facts { display: grid; grid-template-columns: repeat(3, auto); justify-content: space-between; gap: 8px; margin: 0; }
.top__facts dt { font-size: 11px; color: rgb(var(--v-theme-ink3)); }
.top__facts dd { margin: 0; font-size: 13px; font-weight: 650; color: rgb(var(--v-theme-ink)); font-variant-numeric: tabular-nums; white-space: nowrap; }
@media (max-width: 760px) { .top__list { grid-template-columns: minmax(0, 1fr); } }
</style>
