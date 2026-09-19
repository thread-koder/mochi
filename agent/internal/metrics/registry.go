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

var httpLabelNames = append(append([]string{}, labelNames...), "method", "route")

var httpCounterLabelNames = append(append([]string{}, httpLabelNames...), "status_class")

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

// HTTPSeriesKey is one hop RED series (identity + method + route).
type HTTPSeriesKey struct {
	SeriesKey
	Method string
	Route  string
}

func (k HTTPSeriesKey) httpLabelValues() []string {
	return append(k.SeriesKey.labelValues(), k.Method, k.Route)
}

func (k HTTPSeriesKey) counterLabelValues(statusClass string) []string {
	return append(k.httpLabelValues(), statusClass)
}

type Registry struct {
	ConnectsTotal     *prometheus.CounterVec
	ActiveConnections *prometheus.GaugeVec
	TxBytesTotal      *prometheus.CounterVec
	RxBytesTotal      *prometheus.CounterVec

	HTTPRequestsTotal *prometheus.CounterVec
	HTTPDuration      *prometheus.HistogramVec
}

// NewRegistry registers mochi_net_* and mochi_http_* vectors.
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
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mochi_http_requests_total",
				Help: "Client-outbound HTTP/1 request count by status class",
			}, httpCounterLabelNames),
		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:                        "mochi_http_request_duration_seconds",
				Help:                        "Client-outbound HTTP/1 request duration",
				NativeHistogramBucketFactor: 1.1,
			}, httpLabelNames),
	}

	prometheus.MustRegister(
		registry.ConnectsTotal,
		registry.ActiveConnections,
		registry.TxBytesTotal,
		registry.RxBytesTotal,
		registry.HTTPRequestsTotal,
		registry.HTTPDuration,
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

func (r *Registry) RecordHTTP(key SeriesKey, method, route, statusClass string, seconds float64) {
	httpKey := HTTPSeriesKey{SeriesKey: key, Method: method, Route: route}
	r.HTTPRequestsTotal.WithLabelValues(httpKey.counterLabelValues(statusClass)...).Inc()
	r.HTTPDuration.WithLabelValues(httpKey.httpLabelValues()...).Observe(seconds)
}
