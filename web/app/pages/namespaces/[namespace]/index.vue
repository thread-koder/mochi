<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-6">
      <UiBreadcrumb
        :items="breadcrumbs"
      />
      <h1 class="text-4xl font-bold font-heading">
        {{ namespace.name }}
      </h1>
    </div>

    <UiTabs
      v-model="activeTab"
      :tabs="tabs"
    >
      <template #overview>
        <NamespaceOverviewTab :namespace="namespace" />
      </template>
      <template #compute>
        <NamespaceComputeTab
          :namespace="namespace.name"
          :is-active="activeTab === 'compute'"
        />
      </template>
    </UiTabs>
  </div>
</template>

<script setup lang="ts">
import type { NamespaceResponse } from '#shared/types/namespace'

const { namespace: namespaceName } = useRoute().params

const { data: namespaceData, error } = await useApiData<NamespaceResponse>(
  `/api/v1/namespaces/${namespaceName}`,
)

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

if (!namespaceData.value) {
  throw createError({
    statusCode: 404,
    message: 'Namespace not found',
    fatal: true,
  })
}
const namespace = namespaceData.value

const breadcrumbs = [
  { label: 'Home', to: '/' },
  { label: 'Namespaces', to: `/namespaces/${namespace.name}` },
  { label: namespace.name },
]

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'compute', label: 'Compute' },
]
const activeTab = ref('overview')
</script>
