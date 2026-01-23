<template>
  <div class="p-8">
    <!-- Header -->
    <div class="mb-8">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h1 class="text-4xl font-bold font-heading mb-2">
            {{ data?.name }}
          </h1>
          <div class="flex items-center space-x-4">
            <span class="text-sm text-on-surface-secondary">Namespace</span>
            <span
              class="px-3 py-1 rounded-full text-xs border"
              :class="phaseBadgeClass"
            >
              {{ data?.phase }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="mb-6">
      <div class="flex space-x-4 border-b border-primary/20">
        <button
          class="px-4 py-2 cursor-pointer transition-all"
          :class="activeTab === 'overview' ? 'border-b-2 border-primary-light text-primary-light font-semibold' : 'text-on-surface-muted hover:text-on-surface-subtle'"
          @click="setTab('overview')"
        >
          Overview
        </button>
        <button
          class="px-4 py-2 cursor-pointer transition-all"
          :class="activeTab === 'compute' ? 'border-b-2 border-primary-light text-primary-light font-semibold' : 'text-on-surface-muted hover:text-on-surface-subtle'"
          @click="setTab('compute')"
        >
          Compute
        </button>
      </div>

      <!-- Overview Tab -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 scale-90"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-90"
      >
        <NamespaceOverviewTab
          v-show="activeTab === 'overview'"
          :ns-data="data"
        />
      </Transition>

      <!-- Compute Tab -->
      <Transition
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 scale-90"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-90"
      >
        <NamespaceComputeTab
          v-show="activeTab === 'compute'"
          :namespace="data?.name ?? ''"
          :is-active="activeTab === 'compute'"
        />
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

const { namespace } = useRoute().params
const { activeTab, setTab } = useTab('overview')

const { data, error } = await useApiData<Namespace.NamespaceResponse>(`/api/v1/namespaces/${namespace}`)

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
