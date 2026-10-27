<!-- app/components/layout/ContactBody.vue -->
<!-- Mounted only while the dialog is open, so the address is requested when someone asks for it. -->
<template>
  <div class="pane-body contact">
    <p class="contact__lead">
      {{ t('contact.lead') }}
    </p>
    <div class="contact__row">
      <v-icon
        icon="mdi-email-outline"
        size="18"
        aria-hidden="true"
      />
      <div class="contact__body">
        <span class="contact__label">{{ t('contact.email') }}</span>
        <span
          v-if="unavailable"
          class="contact__value contact__value--muted"
        >{{ t('errors.contact_unavailable') }}</span>
        <a
          v-else
          class="contact__value"
          :class="{ 'contact__value--hidden': !email }"
          :href="email ? `mailto:${email}` : undefined"
          :tabindex="email ? undefined : -1"
          :aria-hidden="email ? undefined : 'true'"
        >{{ email || PLACEHOLDER }}</a>
      </div>
      <v-btn
        :class="{ 'contact__copy--hidden': !email }"
        :icon="copied ? 'mdi-check' : 'mdi-content-copy'"
        variant="text"
        size="small"
        :tabindex="email ? undefined : -1"
        :aria-hidden="email ? undefined : 'true'"
        :aria-label="t('contact.copy')"
        :title="copied ? t('contact.copied') : t('contact.copy')"
        @click="copy(email)"
      />
    </div>
    <!-- Outside the row: Cloudflare keeps a hidden frame here while it checks, which must not move the address. -->
    <div
      :id="CONTACT_HOST"
      class="contact__check"
    />
    <span
      class="sr-only"
      role="status"
    >{{ copied ? t('contact.copied') : '' }}</span>
  </div>
</template>

<script setup lang="ts">
import { CONTACT_HOST, useContact } from '~/composables/useContact'

const { t } = useI18n()
const { email, resolved } = useContact()
const { copied, copy } = useCopy()
// No real address ships in the bundle. Letters, unlike asterisks, blur into the same shape as an address.
const PLACEHOLDER = 'username@example.com'
const unavailable = computed(() => resolved.value && !email.value)
</script>

<style scoped>
.contact__lead { margin: 0 0 4px; font-size: 14px; line-height: 1.55; color: rgb(var(--v-theme-ink2)); }
.contact__row { display: grid; grid-template-columns: 18px minmax(0, 1fr) auto; align-items: center; gap: 14px; min-height: 64px; padding: 10px 0; color: rgb(var(--v-theme-ink3)); }
.contact__body { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.contact__label { font-size: 12px; line-height: 16px; color: rgb(var(--v-theme-ink3)); }
/* One fixed line box: the placeholder, the address and the error all take the same height. */
.contact__value {
  display: block;
  height: 22px;
  overflow: hidden;
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
  color: rgb(var(--v-theme-accent));
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: filter 0.45s ease, opacity 0.45s ease;
}
.contact__value--hidden { filter: blur(5px); opacity: 0.75; color: rgb(var(--v-theme-ink3)); user-select: none; pointer-events: none; }
.contact__value--muted { font-size: 14px; font-weight: 400; color: rgb(var(--v-theme-ink3)); }
.contact__copy--hidden { opacity: 0; pointer-events: none; }
@media (prefers-reduced-motion: reduce) { .contact__value { transition: none; } }
</style>
