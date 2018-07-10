/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
