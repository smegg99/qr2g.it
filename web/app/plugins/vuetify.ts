// app/plugins/vuetify.ts
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { en, pl } from 'vuetify/locale'
import { vuetifyOptions } from '~/theme/themes'

export default defineNuxtPlugin((app) => {
  const vuetify = createVuetify({
    ...vuetifyOptions,
    locale: { locale: 'en', fallback: 'en', messages: { en, pl } },
  })
  app.vueApp.use(vuetify)
})
