// app/plugins/api.client.ts
import { client } from '~/generated/api/client.gen'

// The generated client needs its base URL at runtime. Set once here so no page can point elsewhere.
export default defineNuxtPlugin(() => {
  client.setConfig({ baseUrl: String(useRuntimeConfig().public.apiBase) })
})
