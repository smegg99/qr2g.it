<!-- app/pages/docs.vue -->
<template>
  <div class="docs-page site-width">
    <Pane>
      <div class="pane-body docs-body">
        <header class="docs-heading">
          <h1>{{ t('docs.title') }}</h1>
          <p class="docs-lead">
            {{ t('docs.intro') }}
          </p>
        </header>
        <nav
          class="docs-nav"
          :aria-label="t('docs.contents')"
        >
          <a
            v-for="section in sections"
            :key="section"
            :href="`#${section}`"
          >{{ t(`docs.${section}.title`) }}</a>
        </nav>
        <section
          id="short"
          class="docs-section"
        >
          <h2>{{ t('docs.short.title') }}</h2>
          <p>{{ t('docs.short.text') }}</p>
          <CodeExample :code="CURL.short" />
          <p class="response-label">
            {{ t('docs.response') }}
          </p>
          <pre class="response"><code>https://qr2g.it/2</code></pre>
          <p>{{ t('docs.short.explicit') }}</p>
          <CodeExample :code="CURL.text" />
        </section>
        <section
          id="inspect"
          class="docs-section"
        >
          <h2>{{ t('docs.inspect.title') }}</h2>
          <p>{{ t('docs.inspect.text') }}</p>
          <CodeExample :code="CURL.metadata" />
          <p>{{ t('docs.inspect.info') }}</p>
          <CodeExample :code="CURL.info" />
        </section>
        <section
          id="resolve"
          class="docs-section"
        >
          <h2>{{ t('docs.resolve.title') }}</h2>
          <p>{{ t('docs.resolve.text') }}</p>
          <CodeExample :code="CURL.resolve" />
          <p>{{ t('docs.resolve.redirect') }}</p>
          <CodeExample :code="CURL.redirect" />
        </section>
        <section
          id="create"
          class="docs-section"
        >
          <h2>{{ t('docs.create.title') }}</h2>
          <p>{{ t('docs.create.text') }}</p>
          <p>{{ t('docs.create.verification') }}</p>
          <CodeExample :code="CURL.create" />
          <p>{{ t('docs.create.result') }}</p>
        </section>
        <section
          id="limits"
          class="docs-section"
        >
          <h2>{{ t('docs.limits.title') }}</h2>
          <p>{{ t('docs.limits.text') }}</p>
          <ul class="docs-list">
            <li>{{ t('docs.limits.shorthand') }}</li>
            <li>{{ t('docs.limits.post') }}</li>
            <li>{{ t('docs.limits.known') }}</li>
          </ul>
          <p>{{ t('docs.limits.errors') }}</p>
          <div class="status-table-wrap">
            <table class="status-table">
              <caption class="sr-only">
                {{ t('docs.limits.statuses') }}
              </caption>
              <thead>
                <tr>
                  <th scope="col">
                    HTTP
                  </th><th scope="col">
                    {{ t('docs.limits.meaning') }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="status in statuses"
                  :key="status"
                >
                  <th scope="row">
                    <code>{{ status }}</code>
                  </th><td>{{ t(`docs.status.${status}`) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="mt-5">
            <a
              href="/swagger/index.html"
              class="text-accent"
            >{{ t('docs.swagger') }}</a>
          </p>
        </section>
      </div>
    </Pane>
  </div>
</template>

<script setup lang="ts">
import Pane from '~/components/common/Pane.vue'
import CodeExample from '~/components/docs/CodeExample.vue'
import { CURL } from '~/content/curl'

const { t } = useI18n()
const sections = ['short', 'inspect', 'resolve', 'create', 'limits'] as const
const statuses = ['200', '201', '302', '400', '403', '404', '410', '429', '503'] as const
useSeoMeta({ title: () => `${t('docs.title')} | qr2g.it`, description: () => t('docs.intro') })
</script>
