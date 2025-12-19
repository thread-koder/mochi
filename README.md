# Mochi 🍡

> ⚠️ **Early Development**: Mochi is currently in active development. First release (v0.1.0) expected soon. Not production-ready yet.

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.34+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

**Next-generation Kubernetes resource optimization platform** that combines monitoring, optimization, cost analysis, and autoscaling into a single, unified tool. Mochi goes beyond simple requests and limits, offering multi-dimensional optimization across compute, network, storage, and security.

## 🎯 Why Mochi?

Existing Kubernetes optimization tools are fragmented, you need separate tools for monitoring, cost analysis, autoscaling, and optimization. Mochi unifies all of this into one intelligent platform that:

- **Eliminates tool fragmentation** - One tool for everything
- **Goes beyond requests/limits** - Multi-dimensional optimization
- **Predicts before problems** - Statistical and ML-based forecasting
- **Optimizes intelligently** - Network, storage, security-aware
- **Automates safely** - Auto-pilot mode with safety mechanisms

## ✨ Key Features

### Intelligent Resource Analyzer

- Deep pod-level analysis (CPU, memory, network, I/O)
- Dependency mapping between services
- Bottleneck identification
- Resource waste detection (over-provisioned, idle resources)

### Predictive Autoscaling

- Predict demand spikes (deployments, traffic patterns, cron jobs)
- Pre-scale before demand hits
- Multi-metric scaling (not just CPU/memory)
- Cost-aware scaling (spot instances, reserved capacity)

### Cost Intelligence

- Real-time cost attribution (namespace, team, service)
- Cost forecasting
- Savings opportunities (rightsizing, spot usage, reserved instances)
- Multi-cloud cost comparison

### Network Optimization

- Service mesh optimization (Istio, Cilium, Linkerd)
- Pod affinity/anti-affinity recommendations
- Network topology optimization
- Traffic pattern analysis

### Storage Optimization

- PV/PVC usage analysis
- Storage tier recommendations
- Snapshot lifecycle management
- Storage cost optimization

### Security-Aware Optimization

- Security context analysis
- RBAC optimization
- Network policy recommendations
- Compliance checking

### Nice Features

- **"What-if" Scenario Engine** - Simulate changes before applying
- **Auto-Pilot Mode** - Fully automated optimization with safety limits
- **Multi-Cluster Intelligence** - Cross-cluster learning and federated optimization
- **Developer-Friendly** - GitOps integration, PR comments, CI/CD integration
- **Observability-First** - Built-in observability, audit trails, custom dashboards

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Prometheus Ecosystem                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ kube-state-  │  │   cAdvisor   │  │   node-      │     │
│  │   metrics    │  │              │  │  exporter   │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                 │                 │              │
│         └─────────────────┼─────────────────┘              │
│                           │                                │
│                    ┌──────▼──────┐                         │
│                    │ Prometheus  │                         │
│                    └──────┬──────┘                         │
└───────────────────────────┼─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Mochi Controller (Go)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  Analyzer    │  │  Optimizer   │  │ Recommender  │        │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘        │
│         │                 │                 │                 │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐        │
│  │   Enforcer   │  │  Cost Engine │  │  Network     │        │
│  └──────────────┘  └──────────────┘  │  Analyzer    │        │
│                                       └──────────────┘        │
└──────────────┬───────────────────────────────┬───────────────┘
               │                               │
               ▼                               ▼
    ┌──────────────────┐            ┌──────────────────┐
    │   PostgreSQL     │            │      Redis       │
    │ - Recommendations│            │ - Cache          │
    │ - Costs          │            │ - Real-time      │
    │ - History        │            │ - Sessions       │
    │ - Configs        │            └──────────────────┘
    │ - Audit trails   │
    └──────────────────┘
```

## 🛠️ Tech Stack

- **Language**: Go (everything built in Go)
- **Metrics**: Prometheus, kube-state-metrics, cAdvisor, node-exporter
- **Storage**: PostgreSQL (non-metric data), Redis (caching), Prometheus TSDB (metrics)
- **Kubernetes**: client-go, controller-runtime
- **API**: Gin/Echo (REST), gRPC (internal)
- **Messaging**: NATS/JetStream (event streaming)

## 📊 Current Status

### ✅ Completed

- Project planning and architecture design
- Feature specification

## 🤝 Contributing

Contributions are welcome! Mochi is in early development, so there are many opportunities to contribute:

- Bug reports
- Feature suggestions
- Documentation improvements
- Code contributions
- Testing

Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📝 License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with ❤️ for the Kubernetes community
- Inspired by the need for unified optimization tools
- Thanks to all contributors and early adopters

## 📧 Contact & Support

- [Issue Tracker](https://github.com/thread-koder/mochi/issues)
- [Discussions](https://github.com/thread-koder/mochi/discussions)
- Email: [mohamad.fberjawi@gmail.com](mailto:mohamad.fberjawi@gmail.com)
