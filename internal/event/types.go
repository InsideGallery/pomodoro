package event

import "time"

type Type int

const (
	FocusStarted Type = iota
	FocusCompleted
	BreakStarted
	BreakCompleted
	LongBreakStarted
	LongBreakCompleted
	Paused
	Resumed
	Reset
	Tick
	ConfigChanged
)

func (t Type) String() string {
	switch t {
	case FocusStarted:
		return "FocusStarted"
	case FocusCompleted:
		return "FocusCompleted"
	case BreakStarted:
		return "BreakStarted"
	case BreakCompleted:
		return "BreakCompleted"
	case LongBreakStarted:
		return "LongBreakStarted"
	case LongBreakCompleted:
		return "LongBreakCompleted"
	case Paused:
		return "Paused"
	case Resumed:
		return "Resumed"
	case Reset:
		return "Reset"
	case Tick:
		return "Tick"
	case ConfigChanged:
		return "ConfigChanged"
	default:
		return "Unknown"
	}
}

type Event struct {
	Type Type
	Time time.Time
	Data any
}
