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
          {{ namespace }}
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
          <div class="flex items-center space-x-2 flex-wrap">
            <span class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
              {{ data?.type }}
            </span>
            <span
              v-if="data?.type !== 'Pod'"
              class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30"
            >
              {{ data?.ready }}/{{ data?.replicas }} Replicas
            </span>
            <span class="px-3 py-1 rounded-full text-xs font-medium bg-primary/20 text-primary-light border border-primary/30">
              Created {{ timeAgo(data?.created_at ?? '') }}
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
        <WorkloadOverviewTab
          v-show="activeTab === 'overview'"
          :workload-data="data"
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
        <WorkloadComputeTab
          v-show="activeTab === 'compute'"
          :namespace="data?.namespace ?? ''"
          :workload-type="data?.type ?? ''"
          :workload-name="data?.name ?? ''"
          :is-active="activeTab === 'compute'"
        />
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { WorkloadResponse } from '#shared/types/workload'

const route = useRoute()
const { namespace, type, name } = route.params
const { activeTab, setTab } = useTab('overview')

const { data, error } = await useApiData<WorkloadResponse>(`/api/v1/workloads/${namespace}/${type}/${name}`)

const { parseError } = useApiError()
if (error.value) {
  const errorInfo = parseError(error.value, 'Failed to load workload data')
  throw createError({
    status: errorInfo.status,
    statusText: errorInfo.statusText,
    message: errorInfo.message,
    fatal: true,
  })
}
</script>
