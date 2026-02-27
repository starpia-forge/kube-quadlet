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

type KubeUnit struct {
	Unit    UnitSection
	Kube    KubeSection
	Service ServiceSection
	Install InstallSection
}

type NetworkUnit struct {
	Unit    UnitSection
	Network NetworkSection
	Service ServiceSection
	Install InstallSection
}

type ImageUnit struct {
	Unit    UnitSection
	Image   ImageSection
	Service ServiceSection
	Install InstallSection
}

type BuildUnit struct {
	Unit    UnitSection
	Build   BuildSection
	Service ServiceSection
	Install InstallSection
}

type ArtifactUnit struct {
	Unit     UnitSection
	Artifact ArtifactSection
	Service  ServiceSection
	Install  InstallSection
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
	AddCapability   []string
	DropCapability  []string
	NoNewPrivileges bool
	RunInit         bool
	ReadOnly        bool

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

type KubeSection struct {
	Yaml                string
	AutoUpdate          []string
	ConfigMap           []string
	ContainersConfModule []string
	ExitCodePropagation string
	GlobalArgs          []string
	KubeDownForce       bool
	LogDriver           string
	Network             []string
	PodmanArgs          []string
	PublishPort         []string
	SetWorkingDirectory string
	UserNS              string
}

type NetworkSection struct {
	ContainersConfModule []string
	DisableDNS           bool
	DNS                  []string
	Driver               string
	Gateway              []string
	GlobalArgs           []string
	InterfaceName        string
	Internal             bool
	IPAMDriver           string
	IPRange              []string
	IPv6                 bool
	Label                map[string]string
	NetworkDeleteOnStop  bool
	NetworkName          string
	Options              []string
	PodmanArgs           []string
	Subnet               []string
}

type ImageSection struct {
	AllTags              bool
	Arch                 string
	AuthFile             string
	CertDir              string
	ContainersConfModule []string
	Creds                string
	DecryptionKey        string
	GlobalArgs           []string
	Image                string
	ImageTag             string
	OS                   string
	PodmanArgs           []string
	Policy               string
	Retry                int
	RetryDelay           string
	TLSVerify            bool
	Variant              string
}

type BuildSection struct {
	Annotation           map[string]string
	Arch                 string
	AuthFile             string
	BuildArg             map[string]string
	ContainersConfModule []string
	DNS                  []string
	DNSOption            []string
	DNSSearch            []string
	Environment          map[string]string
	File                 string
	ForceRM              bool
	GlobalArgs           []string
	GroupAdd             []string
	IgnoreFile           string
	ImageTag             []string
	Label                map[string]string
	Network              []string
	PodmanArgs           []string
	Pull                 string
	Retry                int
	RetryDelay           string
	Secret               []string
	SetWorkingDirectory  string
	Target               string
	TLSVerify            bool
	Variant              string
	Volume               []string
}

type ArtifactSection struct {
	Artifact             string
	AuthFile             string
	CertDir              string
	ContainersConfModule []string
	Creds                string
	DecryptionKey        string
	GlobalArgs           []string
	PodmanArgs           []string
	Quiet                bool
	Retry                int
	RetryDelay           string
	ServiceName          string
	TLSVerify            bool
}
