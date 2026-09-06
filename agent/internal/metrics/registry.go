package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"
)

var labelNames = []string{
	"src_pod_uid",
	"src_namespace",
	"src_pod",
	"dst_pod_uid",
	"dst_namespace",
	"dst_pod",
	"dst_ip",
	"dst_port",
	"actual_dst_ip",
	"actual_dst_port",
	"protocol",
	"dst_hostname",
}

// SeriesKey is the Prometheus label identifier for one aggregated edge.
type SeriesKey struct {
	SrcPodUID     string
	SrcNamespace  string
	SrcPod        string
	DstPodUID     string
	DstNamespace  string
	DstPod        string
	DstIP         string
	DstPort       int
	ActualDstIP   string
	ActualDstPort int
	Protocol      string
	DstHostname   string
}

// NewSeriesKey builds the Prometheus label set for one client-outbound edge.
// Dest UID or hostname is filled by StampDest after the key is built.
func NewSeriesKey(
	srcUID, srcNS, srcPod, protocol, dstIP string,
	dstPort int,
	actualDstIP string,
	actualDstPort int,
) SeriesKey {
	return SeriesKey{
		SrcPodUID:     srcUID,
		SrcNamespace:  srcNS,
		SrcPod:        srcPod,
		DstIP:         dstIP,
		DstPort:       dstPort,
		ActualDstIP:   actualDstIP,
		ActualDstPort: actualDstPort,
		Protocol:      protocol,
	}
}

func (k SeriesKey) labelValues() []string {
	return []string{
		k.SrcPodUID,
		k.SrcNamespace,
		k.SrcPod,
		k.DstPodUID,
		k.DstNamespace,
		k.DstPod,
		k.DstIP,
		strconv.Itoa(k.DstPort),
		k.ActualDstIP,
		strconv.Itoa(k.ActualDstPort),
		k.Protocol,
		k.DstHostname,
	}
}

type Registry struct {
	ConnectsTotal     *prometheus.CounterVec
	ActiveConnections *prometheus.GaugeVec
	TxBytesTotal      *prometheus.CounterVec
	RxBytesTotal      *prometheus.CounterVec
}

// NewRegistry registers mochi_net_* vectors with the default Prometheus registerer.
func NewRegistry() *Registry {
	registry := &Registry{
		ConnectsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mochi_net_connects_total",
				Help: "Client TCP establishes and first UDP datagrams per flow",
			}, labelNames),
		ActiveConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mochi_net_active_connections",
				Help: "Currently tracked client-outbound L4 flows",
			}, labelNames),
		TxBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mochi_net_tx_bytes_total",
				Help: "Bytes sent on observed client-outbound flows",
			}, labelNames),
		RxBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mochi_net_rx_bytes_total",
				Help: "Bytes received on observed client-outbound flows",
			}, labelNames),
	}

	prometheus.MustRegister(
		registry.ConnectsTotal,
		registry.ActiveConnections,
		registry.TxBytesTotal,
		registry.RxBytesTotal,
	)
	return registry
}

func (r *Registry) AddConnects(key SeriesKey, delta float64) {
	if delta <= 0 {
		return
	}
	r.ConnectsTotal.WithLabelValues(key.labelValues()...).Add(delta)
}

func (r *Registry) SetActive(key SeriesKey, value float64) {
	r.ActiveConnections.WithLabelValues(key.labelValues()...).Set(value)
}

func (r *Registry) AddTxBytes(key SeriesKey, delta float64) {
	if delta <= 0 {
		return
	}
	r.TxBytesTotal.WithLabelValues(key.labelValues()...).Add(delta)
}

func (r *Registry) AddRxBytes(key SeriesKey, delta float64) {
	if delta <= 0 {
		return
	}
	r.RxBytesTotal.WithLabelValues(key.labelValues()...).Add(delta)
}

func (r *Registry) DeleteActive(key SeriesKey) {
	_ = r.ActiveConnections.DeleteLabelValues(key.labelValues()...)
}
