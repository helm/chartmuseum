package event

import (
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
)

var (

	// ChartPushedEventName throws when chart successfully pushed to repo
	ChartPushedEventName = "chart:pushed"
	// ChartDeletedEventName throws when chart deleted from repo
	ChartDeletedEventName = "chart:deleted"
)

type (
	// Event struct defined the action and the additional data from the event
	Event struct {
		Action string
		Logger cm_logger.LoggingFn
		Data   map[string]interface{}
	}
)
