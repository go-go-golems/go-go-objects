package durableobjects

import "time"

const (
	EventDispatchStart = "dispatch.start"
	EventDispatchEnd   = "dispatch.end"
	EventActorStart    = "actor.start"
	EventActorStop     = "actor.stop"
	EventAlarmDispatch = "alarm.dispatch"
	EventEvict         = "actor.evict"
)

type Event struct {
	Name     string
	ID       ObjectID
	Kind     Kind
	Method   string
	Duration time.Duration
	Error    error
	Count    int
}

type EventHook func(Event)
