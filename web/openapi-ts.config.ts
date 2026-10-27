// web/openapi-ts.config.ts
import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  // The bundle, not the source: the source $refs files outside this directory.
  input: '../contracts/backend/dist/openapi.yaml',
  output: 'app/generated/api',
  plugins: ['@hey-api/client-fetch', '@pinia/colada'],
})
