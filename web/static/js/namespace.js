// Alpine.js components
document.addEventListener("alpine:init", () => {
  // Utility functions
  function formatCPU(value) {
    if (value === null || value === undefined) return "N/A";
    const millicores = value * 1000;
    if (value < 1) {
      return Math.round(millicores) + "m";
    }
    return value.toFixed(2) + " cores";
  }

  function formatBytes(value) {
    if (value === null || value === undefined) return "N/A";
    const bytes = value;
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  // Helper function to get CSS variable color, optionally with opacity
  function getCSSVariableColor(variableName, opacity = null) {
    const root = document.documentElement;
    const computedStyle = getComputedStyle(root);
    const value = computedStyle.getPropertyValue(variableName).trim();
    if (!value) return null;

    if (opacity !== null) {
      return value.replace(/\)$/, ` / ${opacity})`);
    }
    return value;
  }

  // Main compute component
  Alpine.data("compute", (namespace) => ({
    namespace: namespace,
    timeRange: "24h",
    customTimeRange: "",
    loading: false,
    error: null,
    analysis: null,

    async loadAnalysis() {
      this.loading = true;
      this.error = null;
      this.analysis = null;

      try {
        const actualTimeRange =
          this.timeRange === "custom"
            ? this.customTimeRange.trim()
            : this.timeRange;
        const url = `/api/v1/compute/analyze/namespaces/${this.namespace}?timeRange=${actualTimeRange}&includeTimeSeries=true&includeWorkloads=true`;
        const response = await fetch(url);
        if (!response.ok) {
          throw new Error(
            await parseErrorResponse(response, "Failed to analyze namespace")
          );
        }
        this.analysis = await response.json();
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },
  }));

  // Time Range Selector component
  Alpine.data("timeRangeSelector", (parent) => ({
    get timeRange() {
      return parent.timeRange;
    },
    set timeRange(value) {
      parent.timeRange = value;
    },
    get customTimeRange() {
      return parent.customTimeRange;
    },
    set customTimeRange(value) {
      parent.customTimeRange = value;
    },

    init() {
      this.$watch("customTimeRange", () => this.updateCustomOptionText(), {
        immediate: true,
      });
      this.$watch("timeRange", () => this.updateCustomOptionText(), {
        immediate: true,
      });
    },

    updateCustomOptionText() {
      if (this.$refs.timeRangeSelect) {
        const customOption = this.$refs.timeRangeSelect.querySelector(
          'option[value="custom"]'
        );
        if (customOption) {
          if (this.customTimeRange.trim()) {
            customOption.textContent = `Custom (${this.customTimeRange.trim()})`;
          } else {
            customOption.textContent = "Custom";
          }
        }
      }
    },

    handleTimeRangeChange() {
      if (this.timeRange !== "custom") {
        this.customTimeRange = "";
        this.updateCustomOptionText();
        parent.loadAnalysis();
      }
    },

    applyCustomTimeRange() {
      if (this.customTimeRange.trim()) {
        this.timeRange = "custom";
        parent.loadAnalysis();
      }
    },
  }));

  // Summary Metrics component
  Alpine.data("summaryMetrics", (parent) => ({
    get utilization() {
      return parent.analysis?.utilization;
    },
    formatCPU: formatCPU,
    formatBytes: formatBytes,
  }));

  // CPU Chart component
  Alpine.data("cpuChart", (parent) => {
    let chart = null;

    return {
      get cpuData() {
        return parent.analysis?.time_series?.cpu;
      },

      destroy() {
        if (chart) {
          chart.destroy();
          chart = null;
        }
      },

      initChart() {
        if (!this.$refs.chart || !this.cpuData || this.cpuData.length === 0)
          return;

        if (chart) {
          chart.destroy();
          chart = null;
        }

        const ctx = this.$refs.chart.getContext("2d");
        // Convert to time-based format
        const data = this.cpuData.map((dp) => ({
          x: new Date(dp.timestamp).getTime(),
          y: dp.value,
        }));
        const maxValue = Math.max(...data.map((d) => d.y));
        const useMillicores = maxValue < 1;

        // Get theme colors
        const primaryColor = getCSSVariableColor("--color-primary-light");
        const primaryColorWithOpacity = getCSSVariableColor(
          "--color-primary-light",
          0.1
        );
        const gridColor = getCSSVariableColor("--color-primary", 0.1);
        const textColor = getCSSVariableColor("--color-on-surface-secondary");

        chart = new Chart(ctx, {
          type: "line",
          data: {
            datasets: [
              {
                label: useMillicores
                  ? "CPU Utilization (millicores)"
                  : "CPU Utilization (cores)",
                data: data,
                parsing: false,
                borderColor: primaryColor,
                backgroundColor: primaryColorWithOpacity,
                borderWidth: 2,
                fill: true,
                tension: 0.6,
                pointRadius: 0,
                pointHoverRadius: 5,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            parsing: false,
            interaction: {
              intersect: false,
              mode: "index",
            },
            plugins: {
              decimation: {
                enabled: true,
                algorithm: "lttb",
                threshold: 300,
              },
              legend: {
                labels: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 14 },
                },
              },
              tooltip: {
                backgroundColor: getCSSVariableColor(
                  "--color-surface-elevated"
                ),
                borderColor: getCSSVariableColor("--color-primary", 0.3),
                borderWidth: 1,
                titleColor: getCSSVariableColor("--color-on-surface-secondary"),
                bodyColor: getCSSVariableColor("--color-on-surface"),
                titleFont: { family: "Inconsolata", size: 14 },
                bodyFont: { family: "Inconsolata", size: 14 },
                cornerRadius: 8,
                padding: 12,
                displayColors: true,
                boxWidth: 12,
                boxHeight: 12,
                boxPadding: 2,
                intersect: false,
                mode: "index",
                callbacks: {
                  title: (context) => {
                    const timestamp = new Date(context[0].parsed.x);
                    return timestamp.toLocaleString("en-US", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                      hour12: false,
                    });
                  },
                  label: (context) => formatCPU(context.parsed.y),
                },
              },
            },
            scales: {
              x: {
                type: "time",
                time: {
                  minUnit: "minute",
                  displayFormats: {
                    millisecond: "HH:mm:ss.SSS",
                    second: "HH:mm:ss",
                    minute: "HH:mm",
                    hour: "MMM d, HH:mm",
                    day: "MMM d, HH:mm",
                    week: "MMM d",
                    month: "MMM yyyy",
                    quarter: "MMM yyyy",
                    year: "yyyy",
                  },
                },
                ticks: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 13 },
                  maxTicksLimit: 10,
                  maxRotation: 0,
                  autoSkip: true,
                },
                grid: { color: gridColor },
                title: {
                  font: { family: "Inconsolata", size: 14 },
                },
              },
              y: {
                ticks: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 13 },
                  callback: (value) => formatCPU(value),
                },
                grid: { color: gridColor },
                title: {
                  display: true,
                  text: useMillicores ? "CPU (millicores)" : "CPU (cores)",
                  color: textColor,
                  font: { family: "Inconsolata", size: 14 },
                },
              },
            },
          },
        });
      },
    };
  });

  // Memory Chart component
  Alpine.data("memoryChart", (parent) => {
    let chart = null;

    return {
      get memoryData() {
        return parent.analysis?.time_series?.memory;
      },

      destroy() {
        if (chart) {
          chart.destroy();
          chart = null;
        }
      },

      initChart() {
        if (
          !this.$refs.chart ||
          !this.memoryData ||
          this.memoryData.length === 0
        )
          return;

        if (chart) {
          chart.destroy();
          chart = null;
        }

        const ctx = this.$refs.chart.getContext("2d");
        // Convert to time-based format
        const data = this.memoryData.map((dp) => ({
          x: new Date(dp.timestamp).getTime(),
          y: dp.value,
        }));

        // Get theme colors
        const secondaryColor = getCSSVariableColor("--color-secondary-light");
        const secondaryColorWithOpacity = getCSSVariableColor(
          "--color-secondary-light",
          0.1
        );
        const gridColor = getCSSVariableColor("--color-secondary", 0.1);
        console.log("Grid color:", gridColor);
        const textColor = getCSSVariableColor("--color-on-surface-secondary");

        chart = new Chart(ctx, {
          type: "line",
          data: {
            datasets: [
              {
                label: "Memory Utilization",
                data: data,
                parsing: false,
                borderColor: secondaryColor,
                backgroundColor: secondaryColorWithOpacity,
                borderWidth: 2,
                fill: true,
                tension: 0.6,
                pointRadius: 0,
                pointHoverRadius: 5,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            parsing: false,
            interaction: {
              intersect: false,
              mode: "index",
            },
            plugins: {
              decimation: {
                enabled: true,
                algorithm: "lttb",
                threshold: 300,
              },
              legend: {
                labels: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 14 },
                },
              },
              tooltip: {
                backgroundColor: getCSSVariableColor(
                  "--color-surface-elevated"
                ),
                borderColor: getCSSVariableColor("--color-secondary", 0.3),
                borderWidth: 1,
                titleColor: getCSSVariableColor("--color-on-surface-secondary"),
                bodyColor: getCSSVariableColor("--color-on-surface"),
                titleFont: { family: "Inconsolata", size: 14 },
                bodyFont: { family: "Inconsolata", size: 14 },
                cornerRadius: 8,
                padding: 12,
                displayColors: true,
                boxWidth: 12,
                boxHeight: 12,
                boxPadding: 2,
                intersect: false,
                mode: "index",
                callbacks: {
                  title: (context) => {
                    const timestamp = new Date(context[0].parsed.x);
                    return timestamp.toLocaleString("en-US", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                      hour12: false,
                    });
                  },
                  label: (context) => formatBytes(context.parsed.y),
                },
              },
            },
            scales: {
              x: {
                type: "time",
                time: {
                  minUnit: "minute",
                  displayFormats: {
                    millisecond: "HH:mm:ss.SSS",
                    second: "HH:mm:ss",
                    minute: "HH:mm",
                    hour: "MMM d, HH:mm",
                    day: "MMM d, HH:mm",
                    week: "MMM d",
                    month: "MMM yyyy",
                    quarter: "MMM yyyy",
                    year: "yyyy",
                  },
                },
                ticks: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 13 },
                  maxTicksLimit: 10,
                  maxRotation: 0,
                  autoSkip: true,
                },
                grid: { color: gridColor },
                title: {
                  font: { family: "Inconsolata", size: 14 },
                },
              },
              y: {
                ticks: {
                  color: textColor,
                  font: { family: "Inconsolata", size: 13 },
                  callback: (value) => formatBytes(value),
                },
                grid: { color: gridColor },
                title: {
                  display: true,
                  text: "Memory",
                  color: textColor,
                  font: { family: "Inconsolata", size: 14 },
                },
              },
            },
          },
        });
      },
    };
  });

  // Workloads List component
  Alpine.data("workloadsList", (namespace, parent) => ({
    namespace: namespace,
    get workloads() {
      return parent.analysis?.workloads;
    },
    sortMetric: "p95",
    sortResource: "cpu",
    filterType: "all",
    filteredWorkloads: [],
    formatCPU: formatCPU,
    formatBytes: formatBytes,

    init() {
      this.$watch(
        "workloads",
        () => {
          if (this.workloads) {
            this.filteredWorkloads = [...this.workloads];
            this.sortWorkloads();
          }
        },
        { immediate: true }
      );
    },

    sortWorkloads() {
      if (!this.filteredWorkloads.length) return;

      this.filteredWorkloads.sort((a, b) => {
        let aValue, bValue;
        const resource = this.sortResource === "cpu" ? "cpu" : "memory";

        if (this.sortMetric === "current") {
          aValue = a.utilization[resource].current;
          bValue = b.utilization[resource].current;
        } else if (this.sortMetric === "p95") {
          aValue = a.utilization[resource].stats.percentile.p95;
          bValue = b.utilization[resource].stats.percentile.p95;
        } else if (this.sortMetric === "mean") {
          aValue = a.utilization[resource].stats.mean;
          bValue = b.utilization[resource].stats.mean;
        } else if (this.sortMetric === "max") {
          aValue = a.utilization[resource].stats.max;
          bValue = b.utilization[resource].stats.max;
        }

        return (bValue || 0) - (aValue || 0);
      });
    },

    filterWorkloads() {
      if (!this.workloads) return;

      if (this.filterType === "all") {
        this.filteredWorkloads = [...this.workloads];
      } else {
        this.filteredWorkloads = this.workloads.filter(
          (w) => w.workload_type.toLowerCase() === this.filterType.toLowerCase()
        );
      }
      this.sortWorkloads();
    },
  }));
});
