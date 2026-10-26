<!-- app/components/home/ResultPanel.vue -->
<template>
  <div class="pane-body">
    <div :class="['display', { 'display--copied': copied }]">
      <button
        type="button"
        class="display__slug"
        :aria-label="`${t('home.result.copy')}: ${link.url}`"
        :title="t('home.result.copy')"
        @click="copy"
      >
        <span class="display__host">{{ parts.host }}/</span>
        <span class="display__code">{{ parts.slug }}</span>
      </button>
      <div class="display__side">
        <dl class="display__rows">
          <div class="display__row">
            <dt>{{ t('home.result.shortLink') }}</dt>
            <dd>
              <a
                :href="link.url"
                target="_blank"
                rel="noopener"
                class="display__url"
              >{{ link.url }}</a>
            </dd>
          </div>
          <div class="display__row">
            <dt>{{ t('home.result.repository') }}</dt>
            <dd>
              <a
                :href="link.currentUrl"
                target="_blank"
                rel="noopener"
                class="display__repo"
                :aria-label="`${t('home.result.openRepo')}: ${repoName}`"
              >
                <v-icon
                  icon="mdi-github"
                  size="16"
                  aria-hidden="true"
                />
                <span>{{ repoName }}</span>
              </a>
            </dd>
          </div>
        </dl>
        <div class="display__keys">
          <v-btn
            :color="copied ? 'success' : 'accent'"
            :prepend-icon="copied ? 'mdi-check' : 'mdi-content-copy'"
            variant="flat"
            class="display__copy"
            @click="copy"
          >
            {{ copied ? t('home.result.copied') : t('home.result.copy') }}
          </v-btn>
          <v-btn
            :href="link.url"
            target="_blank"
            rel="noopener"
            icon="mdi-open-in-new"
            variant="outlined"
            :aria-label="t('home.result.openShort')"
            :title="t('home.result.openShort')"
          />
        </div>
      </div>
      <span
        class="sr-only"
        role="status"
      >{{ copied ? t('home.result.copied') : '' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { LinkResponse } from '~/types/api'

const props = defineProps<{
  link: LinkResponse
}>()

const { t } = useI18n()
const copied = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

const repoName = computed(() => props.link.currentFullName || props.link.initialFullName)

// The slug is the part people read aloud or type, so it is the display; the host is its label.
const parts = computed(() => {
  const bare = props.link.url.replace(/^https?:\/\//i, '').replace(/\/$/, '')
  const cut = bare.indexOf('/')
  return cut < 0 ? { host: bare, slug: props.link.slug } : { host: bare.slice(0, cut), slug: bare.slice(cut + 1) }
})

async function copy() {
  try {
    await navigator.clipboard.writeText(props.link.url)
    copied.value = true
    clearTimeout(timer)
    timer = setTimeout(() => (copied.value = false), 1600)
  }
  catch {
    // Clipboard access is optional in locked-down browser contexts.
  }
}

onBeforeUnmount(() => clearTimeout(timer))
</script>
