package sqlcgen

import "time"

type Device struct {
	ID          string
	DisplayName *string
	Owner       *string
	Location    *string
	Notes       *string
}

type DeviceMetadata struct {
	DeviceID string
	Owner    *string
	Location *string
	Notes    *string
}

type DiscoveryRun struct {
	ID          string
	Status      string
	Scope       *string
	Stats       map[string]any
	StartedAt   time.Time
	CompletedAt *time.Time
	LastError   *string
}

type DiscoveryRunLog struct {
	RunID   string
	Level   string
	Message string
}
