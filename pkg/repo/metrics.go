package repo

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Number of distinct charts
	chartTotalGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "chartmuseum",
			Name:      "charts_served_total",
			Help:      "Current number of charts served",
		},
		[]string{"repo"},
	)
	// Sum of of the number of versions per chart
	chartVersionTotalGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "chartmuseum",
			Name:      "chart_versions_served_total",
			Help:      "Current number of chart versions served",
		},
		[]string{"repo"},
	)
)

func init() {
	prometheus.MustRegister(chartTotalGaugeVec, chartVersionTotalGaugeVec)
}
