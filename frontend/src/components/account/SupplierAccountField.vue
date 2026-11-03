<template>
  <div v-if="supplier || suppliers?.length || modelValue" class="space-y-2">
    <label class="input-label" :for="fieldId">{{ t('admin.suppliers.supplier') }}</label>
    <input v-if="supplier" :id="fieldId" :value="supplier.name" readonly class="input" />
    <select v-else :id="fieldId" :value="modelValue ?? ''" class="input" @change="changeSupplier">
      <option value="">{{ t('admin.suppliers.selfOwned') }}</option>
      <option v-for="item in suppliers" :key="item.id" :value="item.id">{{ item.name }}</option>
    </select>
    <p v-if="selected" class="input-hint">{{ t('admin.suppliers.urlManaged') }} <span class="break-all font-mono">{{ selected.base_url }}</span></p>
    <p v-if="selected" class="input-hint">{{ t('admin.suppliers.keyNameHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Supplier } from '@/api/admin/suppliers'

const props = defineProps<{ modelValue: number | null; suppliers?: Supplier[]; supplier?: Supplier | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()
const { t } = useI18n()
const fieldId = useId()
const selected = computed(() => props.supplier ?? props.suppliers?.find(item => item.id === props.modelValue))
function changeSupplier(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  emit('update:modelValue', value ? Number(value) : null)
}
</script>
