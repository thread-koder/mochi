<template>
  <div class="glass rounded-xl p-6">
    <h2 class="text-2xl font-bold font-heading mb-4">
      System Health
    </h2>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div
        v-for="check in healthIndicators"
        :key="check.name"
        class="flex items-center space-x-3 p-4 rounded-lg"
        :class="check.containerClass"
      >
        <Icon
          :name="check.iconName"
          class="text-xl"
          :class="check.iconClass"
        />
        <div>
          <p class="font-medium">
            {{ check.name }}
          </p>
          <p class="text-sm text-on-surface-secondary">
            {{ check.statusText }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type * as Home from '#shared/types/home'

const props = defineProps<{
  healthChecks?: Home.HomeResponse['health_checks']
}>()

const healthIndicators = computed(() => {
  const checks = [
    { name: 'Database', key: 'database' },
    { name: 'Kubernetes', key: 'kubernetes' },
    { name: 'Prometheus', key: 'prometheus' },
    { name: 'Redis', key: 'redis' },
  ] as const

  return checks.map((check) => {
    const status = props.healthChecks?.[check.key] ?? false
    return {
      name: check.name,
      status,
      iconName: status ? 'lucide:circle-check' : 'lucide:circle-x',
      statusText: status ? 'Connected' : 'Disconnected',
      containerClass: status
        ? 'bg-success/10 border border-success/20'
        : 'bg-error/10 border border-error/20',
      iconClass: status ? 'text-success-light' : 'text-error-light',
    }
  })
})
</script>
