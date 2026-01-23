export const useTab = (initialTab: string) => {
  const activeTab = ref(initialTab)

  const setTab = (tab: string) => {
    activeTab.value = tab
  }

  return {
    activeTab: readonly(activeTab),
    setTab,
  }
}
