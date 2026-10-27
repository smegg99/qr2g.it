// app/composables/useGeneratorFlow.ts
// Orchestrates the home page: repository input, then the resolved link with its design, export and details.
import { useDesignStore } from '~/stores/design'
import { importDesign, type DesignProblem } from '~/utils/design'
import { parseRepoInput } from '~/utils/repository'
import { useGenerator } from './useGenerator'
import { useGeneratorApi } from './useGeneratorApi'

export function useGeneratorFlow() {
  const route = useRoute()
  const designs = useDesignStore()
  const generator = useGenerator(useGeneratorApi())
  const { link, outcome, stage, reset } = generator

  const rawUrl = ref('')
  const problems = ref<DesignProblem[]>([])
  // A message key, translated where it renders so it follows a language change.
  const announcement = ref('')
  let resolvedInput = ''

  onMounted(() => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(route.query)) {
      if (typeof value === 'string') params.append(key, value)
    }
    if ([...params.keys()].length === 0) return
    const imported = importDesign(params, designs.design)
    designs.apply(imported.design)
    problems.value = imported.problems
    if (!imported.repo) return
    if (parseRepoInput(imported.repo)) rawUrl.value = imported.repo
    else problems.value = [...problems.value, { param: 'repo', reason: 'repository' }]
  })

  // Editing the repository invalidates repository-specific results; the design stays.
  watch(rawUrl, (value) => {
    if (link.value && stage.value === 'idle' && value.trim() !== resolvedInput) reset()
  })

  function opened(kind: 'created' | 'existing', input: string) {
    resolvedInput = input.trim()
    announcement.value = `generator.outcome.${kind}`
  }

  async function submit(value: string) {
    rawUrl.value = value
    announcement.value = ''
    const result = await generator.create(value)
    if (result) opened(outcome.value === 'created' ? 'created' : 'existing', value)
  }

  /** Back to the input with the same repository text; the design is kept. */
  function close() {
    reset()
    resolvedInput = ''
    nextTick(() => document.getElementById('repository-input')?.focus())
  }

  return { ...generator, rawUrl, problems, announcement, submit, close }
}
