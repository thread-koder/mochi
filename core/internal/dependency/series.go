package dependency

// ConnectionSeries is one client-outbound connection aggregate matching the mochi_net_* label set.
type ConnectionSeries struct {
	SrcPodUID         string
	SrcNamespace      string
	SrcPod            string
	DstIP             string
	DstPort           int
	ActualDstIP       string
	ActualDstPort     int
	Protocol          string
	Connects          float64
	TxBytes           float64
	RxBytes           float64
	ActiveConnections float64
}
