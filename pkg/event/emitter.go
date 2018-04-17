package event

type (
	// IEventEmitter standard interface for event emitter
	IEventEmitter interface {
		Emit(ev interface{})
	}

	// EventEmitter empty struct
	EventEmitter struct{}
)

// Emit event from EventEmitter
func (emitter *EventEmitter) Emit(ev *Event) {
	action := ev.Action
	for _, pair := range Handler.Events {
		if action == pair.Action {
			pair.Handler(ev)
		}
	}
}

// NewEventEmitter create new EventEmitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{}
}
