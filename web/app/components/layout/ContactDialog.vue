<!-- app/components/layout/ContactDialog.vue -->
<template>
  <v-dialog
    v-model="open"
    max-width="520"
    :aria-labelledby="headingId"
  >
    <Pane>
      <PaneHead
        :id="headingId"
        :title="t('contact.title')"
      >
        <template #actions>
          <v-btn
            icon="mdi-close"
            variant="text"
            size="small"
            :aria-label="t('help.close')"
            :title="t('help.close')"
            @click="open = false"
          />
        </template>
      </PaneHead>
      <ContactBody />
    </Pane>
  </v-dialog>
</template>

<script setup lang="ts">
import Pane from '~/components/common/Pane.vue'
import PaneHead from '~/components/common/PaneHead.vue'
import ContactBody from './ContactBody.vue'

const open = defineModel<boolean>({ required: true })
const { t } = useI18n()
const headingId = useId()
</script>
