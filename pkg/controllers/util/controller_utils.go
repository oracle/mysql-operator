package util

import (
	"time"
)

// NoResyncPeriodFunc Returns 0 for resyncPeriod in case resyncing is not needed.
// See: github.com/kubernetes/kubernetes/pkg/controller/controller_utils.go
func NoResyncPeriodFunc() time.Duration {
	return 0
}
