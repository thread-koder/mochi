<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-4">
      <!-- Breadcrumb -->
      <div class="flex items-center space-x-2 text-sm mb-4">
        <NuxtLink
          to="/"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Home
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <NuxtLink
          :to="`/namespaces/${namespace}`"
          class="text-primary-light hover:text-primary transition-colors"
        >
          Namespaces
        </NuxtLink>
        <Icon
          name="lucide:chevron-right"
          class="text-xs text-on-surface-muted"
        />
        <span class="text-on-surface">{{ data?.name }}</span>
      </div>

      <div class="flex items-center justify-between mb-4">
        <div>
          <h1 class="text-4xl font-bold font-heading mb-2">
            {{ data?.name }}
          </h1>
          <div class="flex items-center space-x-2">
            <span
              class="px-3 py-1 rounded-full text-xs font-medium border"
              :class="phaseBadgeClass"
            >
              {{ data?.phase }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <UiTabs
      v-model="activeTab"
      :tabs="tabs"
    >
      <template #overview>
        <NamespaceOverviewTab :ns-data="data" />
      </template>
      <template #compute>
        <NamespaceComputeTab
          :namespace="data?.name ?? ''"
          :is-active="activeTab === 'compute'"
        />
      </template>
    </UiTabs>
  </div>
</template>

<script setup lang="ts">
import type { NamespaceResponse } from '#shared/types/namespace'

const { namespace } = useRoute().params

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'compute', label: 'Compute' },
]
const activeTab = ref('overview')

const { data, error } = await useApiData<NamespaceResponse>(`/api/v1/namespaces/${namespace}`)

const { parseError } = useApiError()
if (error.value) {
  const errorInfo = parseError(error.value, 'Failed to load namespace data')
  throw createError({
    status: errorInfo.status,
    statusText: errorInfo.statusText,
    message: errorInfo.message,
    fatal: true,
  })
}

const phaseBadgeClass = computed(() => {
  if (data.value?.phase === 'Active') {
    return 'bg-success/20 text-success-light border-success/30'
  }
  return 'bg-on-surface-muted/20 text-on-surface-secondary border-on-surface-muted/30'
})
</script>
