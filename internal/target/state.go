package target

import "time"

type TargetState struct {
	Healthy             bool
	ConsecutiveFailures int
	LastChecked         time.Time
}
