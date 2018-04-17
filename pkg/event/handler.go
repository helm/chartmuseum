package event

import (
	"fmt"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/webhook"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	"github.com/kubernetes-helm/chartmuseum/pkg/integration"
)

var Handler *EventHandler

type (

	// IEventHandler standard interface for event subscription
	IEventHandler interface {
		On(string, interface{})
	}

	// EventHandler have all subscribed events
	EventHandler struct {
		Events []ActionHandlerPair
	}

	// ActionHandlerPair event name and function handler struct
	ActionHandlerPair struct {
		Action  string
		Handler func(interface{})
	}
)

func init() {
	Handler = &EventHandler{}
	Handler.On(ChartPushedEventName, onChartPushedEvent)
	Handler.On(ChartDeletedEventName, onChartDeletedEvent)
}

// On subscribe to event
func (e *EventHandler) On(action string, handler func(interface{})) {
	e.Events = append(e.Events, ActionHandlerPair{
		Action:  action,
		Handler: handler,
	})
}

func onChartPushedEvent(ev interface{}) {
	event := ev.(*Event)
	db := integration.DAL
	logger := event.Logger
	message := fmt.Sprintf("Received event")
	logger(cm_logger.DebugLevel, message,
		"Event_Name", event.Action,
	)
	chartName := event.Data["name"].(string)
	prefix := event.Data["repo"].(string)
	integrations, _ := db.GetIntegrationsMatchResourceByRegex(prefix, chartName, logger)
	logger(cm_logger.DebugLevel, fmt.Sprintf("Got %d integrations\n", len(integrations)))
	for _, integ := range integrations {
		if matchEventActionWithIntegrationTriggers(event, integ) {
			wh := newWebhook(event, integ, logger)
			wh.SendHook(integ.Spec.URL, integ.Spec.Secret, logger)
		}
	}
}

func onChartDeletedEvent(ev interface{}) {
	event := ev.(*Event)
	db := integration.DAL
	logger := event.Logger
	message := fmt.Sprintf("Received event")
	logger(cm_logger.DebugLevel, message,
		"Event_Name", event.Action,
	)
	chartName := event.Data["name"].(string)
	prefix := event.Data["repo"].(string)
	integrations, _ := db.GetIntegrationsMatchResourceByRegex(prefix, chartName, logger)
	for _, integ := range integrations {
		if matchEventActionWithIntegrationTriggers(event, integ) {
			wh := newWebhook(event, integ, logger)
			wh.SendHook(integ.Spec.URL, integ.Spec.Secret, logger)
		}
	}
}

func newWebhook(ev *Event, in *integration.Integration, logger cm_logger.LoggingFn) webhook.Webhook {
	message := fmt.Sprintf("Preparing webhook from integration")
	logger(cm_logger.DebugLevel, message,
		"Name", in.Metadata.Name,
	)
	return webhook.Webhook{
		Version: webhook.Version,
		Kind:    webhook.Kind,
		Metadata: webhook.WebhookMetadata{
			Action: ev.Action,
			Integration: webhook.WebhookMetadataIntegration{
				ID:   in.Metadata.ID,
				Name: in.Metadata.Name,
			},
			Labels:    in.Metadata.Labels,
			Timestamp: time.Now().String(),
		},
		Spec: webhook.WebhookSpec{
			Resource: webhook.WebhookSpecResource{
				Type: "repo",
				Name: ev.Data["repo"].(string),
			},
			Chart: ev.Data["filename"].(string),
		},
	}
}

func matchEventActionWithIntegrationTriggers(ev *Event, in *integration.Integration) bool {
	action := ev.Action
	triggers := in.Spec.Triggers
	for _, t := range triggers {
		if action == t {
			return true
		}
	}
	return false
}
