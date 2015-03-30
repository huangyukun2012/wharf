package server 
import (
	"time"
)

/*===========The following is for container output==========*/
type Device struct {
	PathOnHost        string `json:"PathOnHost,omitempty" yaml:"PathOnHost,omitempty"`
	PathInContainer   string `json:"PathInContainer,omitempty" yaml:"PathInContainer,omitempty"`
	CgroupPermissions string `json:"CgroupPermissions,omitempty" yaml:"CgroupPermissions,omitempty"`
}

// KeyValuePair is a type for generic key/value pairs as used in the Lxc
// configuration
type KeyValuePair struct {
	Key   string `json:"Key,omitempty" yaml:"Key,omitempty"`
	Value string `json:"Value,omitempty" yaml:"Value,omitempty"`
}

// RestartPolicy represents the policy for automatically restarting a container.
//
// Possible values are:
//
//   - always: the docker daemon will always restart the container
//   - on-failure: the docker daemon will restart the container on failures, at
//                 most MaximumRetryCount times
//   - no: the docker daemon will not restart the container automatically
type RestartPolicy struct {
	Name              string `json:"Name,omitempty" yaml:"Name,omitempty"`
	MaximumRetryCount int    `json:"MaximumRetryCount,omitempty" yaml:"MaximumRetryCount,omitempty"`
}

// HostConfig contains the container options related to starting a container on
// a given host
type HostConfig struct {


	Binds           []string               `json:"Binds,omitempty" yaml:"Binds,omitempty"`
	CapAdd          []string               `json:"CapAdd,omitempty" yaml:"CapAdd,omitempty"`
	CapDrop         []string               `json:"CapDrop,omitempty" yaml:"CapDrop,omitempty"`
	ContainerIDFile string                 `json:"ContainerIDFile,omitempty" yaml:"ContainerIDFile,omitempty"`
	Dns             []string               `json:"Dns,omitempty" yaml:"Dns,omitempty"` // For Docker API v1.10 and above only
	DnsSearch       []string               `json:"DnsSearch,omitempty" yaml:"DnsSearch,omitempty"`
	Devices         []Device               `json:"Devices,omitempty" yaml:"Devices,omitempty"`
	ExtraHosts      []string               `json:"ExtraHosts,omitempty" yaml:"ExtraHosts,omitempty"`
	IpcMode         string                 `json:"IpcMode,omitempty" yaml:"IpcMode,omitempty"`
	Links           []string               `json:"Links,omitempty" yaml:"Links,omitempty"`
	LxcConf         []KeyValuePair         `json:"LxcConf,omitempty" yaml:"LxcConf,omitempty"`
	NetworkMode     string                 `json:"NetworkMode,omitempty" yaml:"NetworkMode,omitempty"`
	PidMode         string                 `json:"PidMode,omitempty" yaml:"PidMode,omitempty"`
	PortBindings    map[Port][]PortBinding `json:"PortBindings,omitempty" yaml:"PortBindings,omitempty"`
	Privileged      bool                   `json:"Privileged,omitempty" yaml:"Privileged,omitempty"`
	PublishAllPorts bool                   `json:"PublishAllPorts,omitempty" yaml:"PublishAllPorts,omitempty"`
	RestartPolicy   RestartPolicy          `json:"RestartPolicy,omitempty" yaml:"RestartPolicy,omitempty"`
	VolumesFrom     []string               `json:"VolumesFrom,omitempty" yaml:"VolumesFrom,omitempty"`
	
    ReadonlyRootfs  bool //For Docker server API v1.17 and above only
	SecurityOpts 	[]string//for docker server api v1.17 and above only
}


type Port string

// State represents the state of a container.
type State struct {

	Error      string    `json:"Error,omitempty" yaml:"Error,omitempty"`
	ExitCode   int       `json:"ExitCode,omitempty" yaml:"ExitCode,omitempty"`
	FinishedAt time.Time `json:"FinishedAt,omitempty" yaml:"FinishedAt,omitempty"`
	OOMKilled  bool      `json:"OOMKilled,omitempty" yaml:"OOMKilled,omitempty"`
	Paused     bool      `json:"Paused,omitempty" yaml:"Paused,omitempty"`
	Pid        int       `json:"Pid,omitempty" yaml:"Pid,omitempty"`
	Running    bool      `json:"Running,omitempty" yaml:"Running,omitempty"`
	StartedAt  time.Time `json:"StartedAt,omitempty" yaml:"StartedAt,omitempty"`
	Restarting bool//for docker server api v1.17 and above
}

// PortBinding represents the host/container port mapping as returned in the
// `docker inspect` json
type PortBinding struct {
	HostIP   string `json:"HostIP,omitempty" yaml:"HostIP,omitempty"`
	HostPort string `json:"HostPort,omitempty" yaml:"HostPort,omitempty"`
}

// PortMapping represents a deprecated field in the `docker inspect` output,
// and its value as found in NetworkSettings should always be nil
type PortMapping map[string]string

// NetworkSettings contains network-related information about a container
type NetworkSettings struct {

	Bridge      string                 `json:"Bridge,omitempty" yaml:"Bridge,omitempty"`
	Gateway     string                 `json:"Gateway,omitempty" yaml:"Gateway,omitempty"`
	IPAddress   string                 `json:"IPAddress,omitempty" yaml:"IPAddress,omitempty"`
	IPPrefixLen int                    `json:"IPPrefixLen,omitempty" yaml:"IPPrefixLen,omitempty"`
	PortMapping map[string]PortMapping `json:"PortMapping,omitempty" yaml:"PortMapping,omitempty"`
	Ports       map[Port][]PortBinding `json:"Ports,omitempty" yaml:"Ports,omitempty"`
	//the following is for docker server api v1.17 and above only
	IPv6Gateway string
	LinkLocalIPv6Address string
	LinkLocalIPv6PrefixLen int
	MacAddress 	string
}

// Config is the list of configuration options used when creating a container.
// Config does not the options that are specific to starting a container on a
// given host.  Those are contained in HostConfig
type Config struct {
	AttachStdin     bool                `json:"AttachStdin,omitempty" yaml:"AttachStdin,omitempty"`
	AttachStdout    bool                `json:"AttachStdout,omitempty" yaml:"AttachStdout,omitempty"`
	AttachStderr    bool                `json:"AttachStderr,omitempty" yaml:"AttachStderr,omitempty"`
	Cmd             []string            `json:"Cmd,omitempty" yaml:"Cmd,omitempty"`
	User            string              `json:"User,omitempty" yaml:"User,omitempty"`
	Memory          int64               `json:"Memory,omitempty" yaml:"Memory,omitempty"`
	MemorySwap      int64               `json:"MemorySwap,omitempty" yaml:"MemorySwap,omitempty"`
	CPUShares       int64               `json:"CpuShares,omitempty" yaml:"CpuShares,omitempty"`
	CPUSet          string              `json:"Cpuset,omitempty" yaml:"Cpuset,omitempty"`
	Env             []string            `json:"Env,omitempty" yaml:"Env,omitempty"`
	Entrypoint      []string            `json:"Entrypoint,omitempty" yaml:"Entrypoint,omitempty"`
	ExposedPorts    map[Port]struct{}   `json:"ExposedPorts,omitempty" yaml:"ExposedPorts,omitempty"`
	Hostname        string              `json:"Hostname,omitempty" yaml:"Hostname,omitempty"`
	Domainname      string              `json:"Domainname,omitempty" yaml:"Domainname,omitempty"`
	Image           string              `json:"Image,omitempty" yaml:"Image,omitempty"`
	PortSpecs       []string            `json:"PortSpecs,omitempty" yaml:"PortSpecs,omitempty"`
	Tty             bool                `json:"Tty,omitempty" yaml:"Tty,omitempty"`
	OpenStdin       bool                `json:"OpenStdin,omitempty" yaml:"OpenStdin,omitempty"`
	StdinOnce       bool                `json:"StdinOnce,omitempty" yaml:"StdinOnce,omitempty"`
	Volumes         map[string]struct{} `json:"Volumes,omitempty" yaml:"Volumes,omitempty"`
	WorkingDir      string              `json:"WorkingDir,omitempty" yaml:"WorkingDir,omitempty"`
	OnBuild         []string            `json:"OnBuild,omitempty" yaml:"OnBuild,omitempty"`
	NetworkDisabled bool                `json:"NetworkDisabled,omitempty" yaml:"NetworkDisabled,omitempty"`
	MacAddress 		string				`json:"MacAddress "`//This server API is 1.17 and above 

	/*this 3 iterms below, is below server API 1.17*/
	VolumesFrom     string              `json:"VolumesFrom,omitempty" yaml:"VolumesFrom,omitempty"`
	SecurityOpts    []string            `json:"SecurityOpts,omitempty" yaml:"SecurityOpts,omitempty"`
	Labels          map[string]string   `json:"Labels,omitempty" yaml:"Labels,omitempty"`
}

// SwarmNode containers information about which Swarm node the container is on
type SwarmNode struct {
	ID     string            `json:"ID,omitempty" yaml:"ID,omitempty"`
	IP     string            `json:"IP,omitempty" yaml:"IP,omitempty"`
	Addr   string            `json:"Addr,omitempty" yaml:"Addr,omitempty"`
	Name   string            `json:"Name,omitempty" yaml:"Name,omitempty"`
	CPUs   int64             `json:"CPUs,omitempty" yaml:"CPUs,omitempty"`
	Memory int64             `json:"Memory,omitempty" yaml:"Memory,omitempty"`
	Labels map[string]string `json:"Labels,omitempty" yaml:"Labels,omitempty"`
}

// Container is the type encompasing everything about a container - its config,
// hostconfig, etc.
type Container struct {
/*the following two iterms is below server API 1.17*/
	Node *SwarmNode `json:"Node,omitempty" yaml:"Node,omitempty"`
	SysInitPath    string `json:"SysInitPath,omitempty" yaml:"SysInitPath,omitempty"`


	AppArmorProfile string `json:"AppArmorProfile`// this is for server API version 1.17 and above
	Config *Config `json:"Config,omitempty" yaml:"Config,omitempty"`
	Created time.Time `json:"Created,omitempty" yaml:"Created,omitempty"`
	Driver         string `json:"Driver,omitempty" yaml:"Driver,omitempty"`
	ExecIDs    []string          `json:"ExecIDs,omitempty" yaml:"ExecIDs,omitempty"`
	HostConfig *HostConfig       `json:"HostConfig,omitempty" yaml:"HostConfig,omitempty"`
	HostnamePath   string `json:"HostnamePath,omitempty" yaml:"HostnamePath,omitempty"`
	HostsPath      string `json:"HostsPath,omitempty" yaml:"HostsPath,omitempty"`
	ID string `json:"Id" yaml:"Id"`
	Image  string  `json:"Image,omitempty" yaml:"Image,omitempty"`
	NetworkSettings *NetworkSettings `json:"NetworkSettings,omitempty" yaml:"NetworkSettings,omitempty"`
	Path string   `json:"Path,omitempty" yaml:"Path,omitempty"`
    MountLabel string //this is for server API version 1.17 and above 
	Name           string `json:"Name,omitempty" yaml:"Name,omitempty"`
    ProcessLabel string //this is for server API version 1.17 and above 
	ResolvConfPath string `json:"ResolvConfPath,omitempty" yaml:"ResolvConfPath,omitempty"`
    RestartCount int //this is for server API version 1.17 and above 
	State  State   `json:"State,omitempty" yaml:"State,omitempty"`
	Volumes    map[string]string `json:"Volumes,omitempty" yaml:"Volumes,omitempty"`
	VolumesRW  map[string]bool   `json:"VolumesRW,omitempty" yaml:"VolumesRW,omitempty"`
    ExecDriver string//this is for server API version 1.17 and above 
}

//PsReq is for the request of ps
type InspectReq struct{
	Id	string
}
