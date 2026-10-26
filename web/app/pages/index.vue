<!-- app/pages/index.vue -->
<template>
  <div class="home-page site-width">
    <Transition
      name="swap"
      mode="out-in"
    >
      <div
        v-if="!link"
        key="start"
        class="home-start"
      >
        <h1 class="home-brand">
          <img
            src="/qr2g.it.svg"
            :alt="t('app.name')"
            class="home-logo"
            width="320"
            height="105"
          >
        </h1>
        <p class="home-lead">
          {{ t('app.tagline') }}
        </p>
        <RepoInputForm
          v-model="rawUrl"
          autofocus
          :loading="loading"
          :disabled="loading"
          :show-invalid="invalidShown"
          @submit="onSubmit"
          @input-clear="clearError"
        />
        <div
          id="qr2g-turnstile-host"
          class="d-flex justify-center"
        />
        <div
          class="home-error"
          role="status"
          aria-live="polite"
        >
          <ErrorMessage
            v-if="error?.code"
            :code="error.code"
            @dismiss="clearError"
          />
        </div>
      </div>
      <HomeResolved
        v-else
        key="result"
        :link="link"
        class="result-pane"
        @close="onClose"
      />
    </Transition>
  </div>
</template>

<script setup lang="ts">
import RepoInputForm from '~/components/home/RepoInputForm.vue'
import ErrorMessage from '~/components/home/ErrorMessage.vue'
import HomeResolved from '~/components/home/HomeResolved.vue'

const { t } = useI18n()
const { link, loading, error, create, reset, clearError } = useShortLink()
const rawUrl = ref('')
const invalidShown = computed(() => error.value?.code === 'invalid_url')

async function onSubmit(value: string) {
  if (loading.value) return
  rawUrl.value = value
  await create({ url: value })
}
function onClose() {
  reset()
  rawUrl.value = ''
}
useSeoMeta({ title: () => t('home.metaTitle'), description: () => t('app.tagline') })
</script>
