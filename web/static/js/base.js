// Global utility for parsing error responses
async function parseErrorResponse(response, defaultMessage = "Request failed") {
  try {
    const errorData = await response.json();
    return errorData.details
      ? `${errorData.error || defaultMessage}: ${errorData.details}`
      : errorData.error || response.statusText || defaultMessage;
  } catch {
    return response.statusText || defaultMessage;
  }
}

// Alpine.js components
document.addEventListener("alpine:init", () => {
  // Global counter component
  Alpine.data("counter", (target, intervalMs = 50) => ({
    count: 0,
    interval: null,
    init() {
      this.interval = setInterval(() => {
        if (this.count < target) {
          this.count++;
        } else {
          clearInterval(this.interval);
          this.interval = null;
        }
      }, intervalMs);
    },
    destroy() {
      if (this.interval) {
        clearInterval(this.interval);
        this.interval = null;
      }
    },
  }));

  // Namespaces navigation items component
  Alpine.data("namespacesNavItems", () => ({
    open: true,
    namespaces: [],
    loading: false,
    error: null,

    init() {
      if (this.open) {
        this.loadNamespaces();
      }
    },

    async loadNamespaces() {
      if (this.namespaces.length > 0) return;
      this.loading = true;
      this.error = null;
      try {
        const response = await fetch("/api/v1/namespaces");
        if (!response.ok) {
          throw new Error(
            await parseErrorResponse(response, "Failed to load namespaces")
          );
        }
        const data = await response.json();
        this.namespaces = data.namespaces || [];
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },

    isActive(namespace) {
      return window.location.pathname === "/namespaces/" + namespace;
    },

    getPhaseColor(phase) {
      switch (phase.toLowerCase()) {
        case "active":
          return "bg-green-400";
        case "terminating":
          return "bg-yellow-400";
        default:
          return "bg-gray-400";
      }
    },
  }));
});
