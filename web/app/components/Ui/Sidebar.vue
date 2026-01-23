<template>
  <aside class="w-64 glass border-r border-primary/30 flex flex-col">
    <!-- Logo -->
    <div class="p-6 border-b border-primary/30 flex items-center justify-between">
      <h1
        class="text-2xl font-bold font-heading bg-linear-to-r from-primary-light
        via-primary to-secondary-light bg-clip-text text-transparent drop-shadow-lg"
      >
        Mochi
      </h1>
      <UiThemeSwitcher />
    </div>

    <!-- Navigation -->
    <nav class="flex-1 p-4 space-y-2 overflow-y-auto">
      <!-- Home Link -->
      <NuxtLink
        to="/"
        class="flex items-center space-x-3 px-4 py-2 rounded-lg transition-all
          text-sm hover:bg-primary/10 hover:text-on-surface text-on-surface-subtle"
        active-class="hover:text-primary-light hover:bg-primary/20
         bg-primary/20 border-l-2 border-primary-light text-primary-light"
        exact-active-class="hover:text-primary-light hover:bg-primary/20
         bg-primary/20 border-l-2 border-primary-light text-primary-light"
      >
        <Icon
          name="lucide:home"
        />
        <span class="font-medium">Home</span>
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
          class="flex items-center justify-between px-4 py-2 rounded-lg transition-all
           text-sm hover:bg-primary/10 hover:text-on-surface-secondary text-on-surface-muted"
          active-class="bg-primary/20 hover:bg-primary/20 border-l-2 border-primary-light
           text-primary-light hover:text-primary-light font-medium"
          exact-active-class="bg-primary/20 hover:bg-primary/20 border-l-2 border-primary-light
           text-primary-light hover:text-primary-light font-medium"
        >
          <span>{{ ns.name }}</span>
          <span
            :class="`w-2 h-2 rounded-full ${phaseColor(ns.phase)}`"
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
          class="block px-4 py-2 rounded-lg transition-all
           text-sm hover:bg-primary/10 hover:text-on-surface-secondary text-on-surface-muted"
          active-class="bg-primary/20 border-l-2 border-primary-light
           text-primary-light hover:bg-primary/20 font-medium"
          exact-active-class="bg-primary/20 border-l-2 border-primary-light
           text-primary-light hover:bg-primary/20 font-medium"
        >
          Compute
        </NuxtLink>
      </UiNavList>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import type * as Namespace from '#shared/types/namespace'

const { parseError } = useApiError()

const { data, pending, error }
  = await useApiData<Namespace.Namespace[]>('/api/v1/namespaces')

const phaseColor = (phase: string) => {
  return phase.toLowerCase() === 'active' ? 'bg-success' : 'bg-warning'
}
</script>
