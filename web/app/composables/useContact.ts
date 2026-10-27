// app/composables/useContact.ts
// The server decides which address a request earned; nothing here caches it.
// A check that fails, or does not finish in time, is sent as empty and earns the catch-all.
import { revealContact } from '~/generated/api/sdk.gen'
import { call } from '~/utils/api'

export const CONTACT_HOST = 'qr2g-contact-turnstile'

// Cloudflare has no deadline of its own; a script that never loads never calls back.
const GIVE_UP_MS = 15_000

export function useContact() {
  const turnstile = useTurnstile()
  const email = ref('')
  const resolved = ref(false)
  let owner: AbortController | null = null

  async function reveal() {
    owner?.abort()
    const request = new AbortController()
    owner = request
    const giveUp = setTimeout(() => request.abort(), GIVE_UP_MS)
    let token = ''
    try {
      token = (await turnstile.getToken(request.signal, { action: 'contact', container: CONTACT_HOST })) ?? ''
    }
    catch {
      // Unverifiable: the server answers an empty token with the catch-all.
    }
    finally {
      clearTimeout(giveUp)
    }
    if (owner !== request) return
    const result = await call(() => revealContact({ body: { turnstileToken: token || undefined } }))
    if (owner !== request) return
    email.value = result.data?.email ?? ''
    resolved.value = true
  }

  onMounted(reveal)
  onBeforeUnmount(() => {
    owner?.abort()
    owner = null
  })

  return { email, resolved }
}
