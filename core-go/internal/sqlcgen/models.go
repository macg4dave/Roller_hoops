package sqlcgen

import "time"

type Device struct {
	ID          string
	DisplayName *string
	Owner       *string
	Location    *string
	Notes       *string
}

type DeviceListItem struct {
	ID           string
	DisplayName  *string
	PrimaryIP    *string
	Owner        *string
	Location     *string
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastSeenAt   *time.Time
	LastChangeAt time.Time
	SortTs       time.Time
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

type DeviceIP struct {
	IP            string
	InterfaceID   *string
	InterfaceName *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DeviceMAC struct {
	MAC           string
	InterfaceID   *string
	InterfaceName *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DeviceInterface struct {
	ID             string
	Name           *string
	Ifindex        *int32
	Descr          *string
	Alias          *string
	MAC            *string
	AdminStatus    *int32
	OperStatus     *int32
	MTU            *int32
	SpeedBps       *int64
	PVID           *int32
	PVIDObservedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DeviceService struct {
	Protocol   *string
	Port       *int32
	Name       *string
	State      *string
	Source     *string
	ObservedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type DeviceSNMP struct {
	DeviceID      string
	Address       *string
	SysName       *string
	SysDescr      *string
	SysObjectID   *string
	SysContact    *string
	SysLocation   *string
	LastSuccessAt *time.Time
	LastError     *string
	UpdatedAt     time.Time
}

type DeviceLink struct {
	ID               string
	LinkKey          string
	PeerDeviceID     string
	LocalInterfaceID *string
	PeerInterfaceID  *string
	LinkType         *string
	Source           string
	ObservedAt       *time.Time
	UpdatedAt        time.Time
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
