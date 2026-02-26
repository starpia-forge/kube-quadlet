package quadlet

type ContainerUnit struct {
	Unit      UnitSection
	Container ContainerSection
	Service   ServiceSection
	Install   InstallSection
}

type PodUnit struct {
	Unit    UnitSection
	Pod     PodSection
	Service ServiceSection
	Install InstallSection
}

type VolumeUnit struct {
	Unit    UnitSection
	Volume  VolumeSection
	Service ServiceSection // Technically Quadlet generates a service for volume
	Install InstallSection
}

type UnitSection struct {
	Description string
	Wants       []string
	Requires    []string
	After       []string
	Before      []string
}

type ServiceSection struct {
	Restart         string
	TimeoutStartSec string
}

type InstallSection struct {
	WantedBy []string
}

type ContainerSection struct {
	Image             string
	Exec              string // Can be multiple words
	Entrypoint        string
	Environment       map[string]string
	EnvironmentFile   []string
	PublishPort       []string
	Volume            []string
	User              string
	Group             string
	WorkingDir        string
	Pod               string
	Network           []string
	NetworkAlias      []string
	HostName          string

	// Health Check
	HealthCmd         string
	HealthInterval    string
	HealthRetries     int
	HealthTimeout     string
	HealthStartPeriod string

	// Resources
	Memory string

	// Security
	AddCapability  []string
	DropCapability []string
	NoNewPrivileges bool
	RunInit        bool
	ReadOnly       bool

	// Metadata
	Label      map[string]string
	Annotation map[string]string

	// Advanced
	PodmanArgs []string
}

type PodSection struct {
	PodName      string
	PublishPort  []string
	Volume       []string
	Network      []string
	NetworkAlias []string
	IP           string
	GlobalArgs   []string
	PodmanArgs   []string
}

type VolumeSection struct {
	VolumeName string
	Label      map[string]string
	User       string
	Group      string
	Driver     string
	Options    []string
}
