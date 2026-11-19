<template>
  <fieldset class="col-span-full space-y-3">
    <label class="flex items-center gap-2 text-sm">
      <Toggle :model-value="!!rule" @update:model-value="toggle" />
      {{ t('admin.scheduledTests.quality.title') }}
    </label>
    <div v-if="rule" class="grid gap-3 sm:grid-cols-2">
      <label class="text-xs sm:col-span-2">
        {{ t('admin.scheduledTests.quality.models') }}
        <input class="input mt-1" :value="rule.models.join(', ')" @change="setModels" />
      </label>
      <label class="text-xs">
        {{ t('admin.scheduledTests.quality.action') }}
        <select v-model="rule.action" class="input mt-1">
          <option v-for="action in actions" :key="action" :value="action">{{ t('admin.scheduledTests.quality.' + action) }}</option>
        </select>
      </label>
      <label v-if="rule.action === 'group'" class="text-xs">
        {{ t('admin.scheduledTests.quality.groupId') }}
        <select v-model.number="rule.group_id" class="input mt-1">
          <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
        </select>
      </label>
      <label v-for="field in numericFields" :key="field.key" class="text-xs">
        {{ t('admin.scheduledTests.quality.' + field.key) }}
        <input v-model.number="rule[field.key]" class="input mt-1" type="number" :min="field.min" :max="field.max" :step="field.step" />
      </label>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { AccountQualityRule } from '@/types'

const props = defineProps<{ model: string; groups: { id: number; name: string }[] }>()
const rule = defineModel<AccountQualityRule | undefined>()
const { t } = useI18n()
const actions = ['observe', 'model', 'group', 'account'] as const
const numericFields = [
  { key: 'failures', min: 1, max: 20, step: 1 },
  { key: 'passes', min: 1, max: 20, step: 1 },
  { key: 'max_ttft_ms', min: 0, max: 300000, step: 1 },
  { key: 'min_tps', min: 0, max: 10000, step: 0.1 },
  { key: 'max_error_percent', min: 0, max: 100, step: 0.1 }
] as const
function toggle(enabled: boolean) {
  rule.value = enabled ? { models: props.model ? [props.model] : [], action: 'observe', group_id: props.groups[0]?.id, failures: 3, passes: 2, max_ttft_ms: 0, min_tps: 0, max_error_percent: 0 } : undefined
}
function setModels(event: Event) {
  if (rule.value) rule.value.models = (event.target as HTMLInputElement).value.split(/[,，\n]/).map(model => model.trim()).filter(Boolean)
}
</script>
