// Package runtime contains small, Portfolio-owned production adapters.
package runtime

import "time"

// Clock returns the current UTC time for Portfolio application operations.
type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }
