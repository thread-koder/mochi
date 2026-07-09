<template>
  <aside class="w-64 bg-surface-elevated flex flex-col">
    <!-- Logo -->
    <div class="px-6 pt-6 pb-4 flex items-center justify-between">
      <h1 class="text-2xl font-bold font-heading text-on-surface">
        Mochi
      </h1>
      <UiThemeSwitcher />
    </div>

    <!-- Navigation -->
    <nav class="flex-1 px-4 pb-4 pt-2 space-y-2 overflow-y-auto">
      <!-- Home Link -->
      <NuxtLink
        to="/"
        class="flex items-center space-x-3 px-4 py-2 rounded-lg text-sm
          text-on-surface-muted hover:bg-primary/10 hover:text-on-surface"
        active-class="bg-primary/10 !text-primary-light
          hover:bg-primary/10 hover:!text-primary-light"
        exact-active-class="bg-primary/10 !text-primary-light
          hover:bg-primary/10 hover:!text-primary-light"
      >
        <Icon
          name="lucide:house"
        />
        <span>Home</span>
      </NuxtLink>

      <!-- Namespaces List -->
      <UiNavList
        title="Namespaces"
        icon="lucide:layers"
        default-open
      >
        <div
          v-if="pending"
          class="text-xs text-on-surface-muted px-4 py-2"
        >
          Loading...
        </div>
        <div
          v-else-if="error"
          class="text-xs text-error-light px-4 py-2"
        >
          {{ parseError(error, 'Failed to load namespaces').message }}
        </div>
        <div
          v-else-if="!data || data.length === 0"
          class="text-xs text-on-surface-muted px-4 py-2"
        >
          No namespaces found
        </div>
        <NuxtLink
          v-for="ns in data || []"
          :key="ns.name"
          :to="`/namespaces/${ns.name}`"
          class="flex items-center justify-between px-4 py-2 rounded-lg text-sm
           text-on-surface-muted hover:bg-primary/10 hover:text-on-surface"
          active-class="bg-primary/10 !text-primary-light
           hover:bg-primary/10 hover:!text-primary-light"
          exact-active-class="bg-primary/10 !text-primary-light
           hover:bg-primary/10 hover:!text-primary-light"
        >
          <span>{{ ns.name }}</span>
          <span
            :class="['w-2 h-2 rounded-full', phaseColor(ns.phase)]"
            :title="ns.phase"
          />
        </NuxtLink>
      </UiNavList>

      <!-- Recommendations List -->
      <UiNavList
        title="Recommendations"
        icon="lucide:lightbulb"
        default-open
      >
        <NuxtLink
          to="/recommendations/compute"
          class="block px-4 py-2 rounded-lg text-sm
           text-on-surface-muted hover:bg-primary/10 hover:text-on-surface"
          active-class="bg-primary/10 !text-primary-light
           hover:bg-primary/10 hover:!text-primary-light"
          exact-active-class="bg-primary/10 !text-primary-light
           hover:bg-primary/10 hover:!text-primary-light"
        >
          Compute
        </NuxtLink>
      </UiNavList>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import type { Namespace } from '#shared/types/namespace'

const { parseError } = useApiError()

const { data, pending, error }
  = await useApiData<Namespace[]>('/api/v1/namespaces')

const phaseColor = (phase: string) => {
  return phase.toLowerCase() === 'active' ? 'bg-success' : 'bg-warning'
}
</script>
