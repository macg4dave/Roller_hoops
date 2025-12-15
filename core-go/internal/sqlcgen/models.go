package sqlcgen

import "time"

type Device struct {
	ID          string
	DisplayName *string
	Owner       *string
	Location    *string
	Notes       *string
}

type DeviceNameCandidate struct {
	DeviceID   string
	Name       string
	Source     string
	Address    *string
	ObservedAt time.Time
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
	ID        int64
	RunID     string
	Level     string
	Message   string
	CreatedAt time.Time
}

type DeviceChangeEvent struct {
	EventID  string
	DeviceID string
	EventAt  time.Time
	Kind     string
	Summary  string
	Details  map[string]any
}

type ListDeviceChangeEventsParams struct {
	BeforeEventAt *time.Time
	BeforeEventID *string
	SinceEventAt  *time.Time
	Limit         int32
}

type ListDeviceChangeEventsForDeviceParams struct {
	DeviceID      string
	BeforeEventAt *time.Time
	BeforeEventID *string
	Limit         int32
}
