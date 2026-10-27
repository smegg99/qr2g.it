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
          <p>{{ t('docs.short.sideEffects') }}</p>
          <p>{{ t('docs.short.qr') }}</p>
          <CodeExample :code="CURL.qr" />
        </section>
        <section
          id="text"
          class="docs-section"
        >
          <h2>{{ t('docs.text.title') }}</h2>
          <p>{{ t('docs.short.explicit') }}</p>
          <CodeExample :code="CURL.text" />
        </section>
        <section
          id="lookup"
          class="docs-section"
        >
          <h2>{{ t('docs.lookup.title') }}</h2>
          <p>{{ t('docs.inspect.lookup') }}</p>
          <CodeExample :code="CURL.lookup" />
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
          id="automation"
          class="docs-section"
        >
          <h2>{{ t('docs.automation.title') }}</h2>
          <p>{{ t('docs.automation.text') }}</p>
          <CodeExample :code="CURL.automation" />
        </section>
        <section
          id="batch"
          class="docs-section"
        >
          <h2>{{ t('docs.batch.title') }}</h2>
          <p>{{ t('docs.automation.batch') }}</p>
          <CodeExample :code="CURL.batch" />
          <p>{{ t('docs.automation.quota') }}</p>
          <CodeExample :code="CURL.usage" />
        </section>
        <section
          id="assets"
          class="docs-section"
        >
          <h2>{{ t('docs.assets.title') }}</h2>
          <p>{{ t('docs.assets.text') }}</p>
          <CodeExample :code="CURL.asset" />
          <p>{{ t('docs.assets.reliability') }}</p>
        </section>
        <section
          id="explore"
          class="docs-section"
        >
          <h2>{{ t('docs.explore.title') }}</h2>
          <p>{{ t('docs.explore.text') }}</p>
          <CodeExample :code="CURL.search" />
          <p>{{ t('docs.explore.stats') }}</p>
          <CodeExample :code="CURL.stats" />
          <p>{{ t('docs.explore.top') }}</p>
          <CodeExample :code="CURL.top" />
        </section>
        <section
          id="prefill"
          class="docs-section"
        >
          <h2>{{ t('docs.prefill.title') }}</h2>
          <p>{{ t('docs.prefill.text') }}</p>
          <CodeExample :code="DESIGN_LINK_EXAMPLE" />
          <p>{{ t('docs.prefill.params') }}</p>
          <p>{{ t('docs.prefill.honest') }}</p>
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
        </section>
        <section
          id="statuses"
          class="docs-section"
        >
          <h2>{{ t('docs.statuses.title') }}</h2>
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
          <ul class="docs-list mt-5">
            <li>
              <a
                href="/api/v1/openapi.yaml"
                class="text-accent"
              >{{ t('docs.openapi') }}</a>
            </li>
            <li>
              <a
                href="/swagger/index.html"
                class="text-accent"
              >{{ t('docs.swagger') }}</a>
            </li>
            <li>
              <a
                href="/guide.md"
                class="text-accent"
              >{{ t('docs.guide') }}</a>
            </li>
          </ul>
        </section>
      </div>
    </Pane>
  </div>
</template>

<script setup lang="ts">
import Pane from '~/components/common/Pane.vue'
import CodeExample from '~/components/docs/CodeExample.vue'
import { CURL, DESIGN_LINK_EXAMPLE } from '~/content/curl'

const { t } = useI18n()
const sections = ['short', 'text', 'lookup', 'inspect', 'resolve', 'create', 'automation', 'batch', 'assets', 'explore', 'prefill', 'limits', 'statuses'] as const
const statuses = ['200', '201', '302', '400', '401', '403', '404', '410', '429', '503'] as const
useSeoMeta({ title: () => `${t('docs.title')} | qr2g.it`, description: () => t('docs.intro') })
</script>
