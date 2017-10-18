package repo

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Number of distinct charts
	chartTotalGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "chartmuseum",
			Name:      "total_charts_served",
			Help:      "Current number of charts served",
		},
	)
	// Sum of of the number of versions per chart
	chartVersionTotalGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "chartmuseum",
			Name:      "total_chart_versions_served",
			Help:      "Current number of chart versions served",
		},
	)
)

func init() {
	prometheus.MustRegister(chartTotalGauge, chartVersionTotalGauge)
}
