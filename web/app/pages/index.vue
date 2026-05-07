<!-- app/pages/index.vue -->
<template>
  <v-container class="home-container" fluid>
    <div class="home-stage">
      <v-fade-transition mode="out-in" :duration="{ enter: 260, leave: 180 }">
        <div v-if="!link" key="form" class="d-flex flex-column align-center ga-5 w-100">
          <HomeHero />

          <RepoInputForm v-model="rawUrl" class="form-wrap" autofocus :loading="loading" :disabled="loading"
            :show-invalid="invalidShown" @submit="onSubmit" @input-clear="clearError" />

          <v-expand-transition>
            <div v-if="error?.code" class="form-wrap">
              <ErrorMessage :code="error.code" @dismiss="clearError" />
            </div>
          </v-expand-transition>
        </div>

        <HomeResolved v-else key="result" :link="link" class="card-wrap" @close="onClose" />
      </v-fade-transition>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import RepoInputForm from '~/components/home/RepoInputForm.vue'
import ErrorMessage from '~/components/home/ErrorMessage.vue'
import HomeHero from '~/components/home/HomeHero.vue'
import HomeResolved from '~/components/home/HomeResolved.vue'

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

useHead({
  title: 'qr2g.it - Permanent GitHub links',
})
</script>

<style scoped>
.home-container {
  display: flex;
  justify-content: center;
  padding: 80px 16px 24px;
}

.home-stage {
  width: 100%;
  max-width: 1040px;
  position: relative;
  z-index: 1;
}

.form-wrap {
  width: min(100%, 720px);
}

.card-wrap {
  width: 100%;
  max-width: 920px;
  margin: 0 auto;
}

@media (min-width: 600px) {
  .home-container {
    padding: 64px 24px 32px;
  }
}

@media (min-width: 960px) {
  .home-container {
    padding: 80px 32px 40px;
  }
}
</style>
