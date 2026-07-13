<template>
  <p
    v-if="allHealthy"
    class="flex items-center gap-1.5 text-xs text-on-surface-muted mt-1"
  >
    <Icon
      name="lucide:circle-check"
      class="text-success-light shrink-0"
    />
    All systems operational
  </p>
  <p
    v-else
    class="flex items-center gap-1.5 text-sm text-error-light mt-1"
  >
    <Icon
      name="lucide:circle-alert"
      class="shrink-0"
    />
    {{ failureMessage }}
  </p>
</template>

<script setup lang="ts">
import type { HomeResponse } from '#shared/types/home'

const props = defineProps<{
  healthChecks: HomeResponse['health_checks']
}>()

const checks = [
  { name: 'Database', key: 'database' },
  { name: 'Kubernetes', key: 'kubernetes' },
  { name: 'Prometheus', key: 'prometheus' },
  { name: 'Redis', key: 'redis' },
] as const

const healthIndicators = computed(() =>
  checks.map(check => ({
    name: check.name,
    status: props.healthChecks[check.key],
  })),
)

const allHealthy = computed(() =>
  healthIndicators.value.every(check => check.status),
)

const failureMessage = computed(() =>
  healthIndicators.value
    .filter(check => !check.status)
    .map(check => `${check.name} disconnected`)
    .join(' · '),
)
</script>
