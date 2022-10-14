package containerlab

type Config struct {
	Name     string   `yaml:"name"`
	Prefix   *string  `yaml:"prefix,omitempty"`
	Mgmt     *MgmtNet `yaml:"mgmt,omitempty"`
	Topology Topology `yaml:"topology"`
}

// MgmtNet struct defines the management network options.
type MgmtNet struct {
	Network string `yaml:"network,omitempty" json:"network,omitempty"` // container runtime network name
	Bridge  string `yaml:"bridge,omitempty" json:"bridge,omitempty"`
	// linux bridge backing the runtime network
	IPv4Subnet     string `yaml:"ipv4_subnet,omitempty" json:"ipv4-subnet,omitempty"`
	IPv4Gw         string `yaml:"ipv4-gw,omitempty" json:"ipv4-gw,omitempty"`
	IPv6Subnet     string `yaml:"ipv6_subnet,omitempty" json:"ipv6-subnet,omitempty"`
	IPv6Gw         string `yaml:"ipv6-gw,omitempty" json:"ipv6-gw,omitempty"`
	MTU            string `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	ExternalAccess *bool  `yaml:"external-access,omitempty" json:"external-access,omitempty"`
}

type Topology struct {
	Defaults *NodeDefinition            `yaml:"defaults,omitempty"`
	Kinds    map[string]*NodeDefinition `yaml:"kinds,omitempty"`
	Nodes    map[string]*NodeDefinition `yaml:"nodes,omitempty"`
	Links    []*LinkConfig              `yaml:"links,omitempty"`
}

type NodeDefinition struct {
	Kind                 string            `yaml:"kind,omitempty"`
	Group                string            `yaml:"group,omitempty"`
	Type                 string            `yaml:"type,omitempty"`
	StartupConfig        string            `yaml:"startup-config,omitempty"`
	StartupDelay         uint              `yaml:"startup-delay,omitempty"`
	EnforceStartupConfig bool              `yaml:"enforce-startup-config,omitempty"`
	Config               *ConfigDispatcher `yaml:"config,omitempty"`
	Image                string            `yaml:"image,omitempty"`
	License              string            `yaml:"license,omitempty"`
	Position             string            `yaml:"position,omitempty"`
	Entrypoint           string            `yaml:"entrypoint,omitempty"`
	Cmd                  string            `yaml:"cmd,omitempty"`
	// list of subject Alternative Names (SAN) to be added to the node's certificate
	SANs []string `yaml:"SANs,omitempty"`
	// list of commands to run in container
	Exec []string `yaml:"exec,omitempty"`
	// list of bind mount compatible strings
	Binds []string `yaml:"binds,omitempty"`
	// list of port bindings
	Ports []string `yaml:"ports,omitempty"`
	// user-defined IPv4 address in the management network
	MgmtIPv4 string `yaml:"mgmt_ipv4,omitempty"`
	// user-defined IPv6 address in the management network
	MgmtIPv6 string `yaml:"mgmt_ipv6,omitempty"`
	// list of ports to publish with mysocketctl
	Publish []string `yaml:"publish,omitempty"`
	// environment variables
	Env map[string]string `yaml:"env,omitempty"`
	// external file containing environment variables
	EnvFiles []string `yaml:"env-files,omitempty"`
	// linux user used in a container
	User string `yaml:"user,omitempty"`
	// container labels
	Labels map[string]string `yaml:"labels,omitempty"`
	// container networking mode. if set to `host` the host networking will be used for this node, else bridged network
	NetworkMode string `yaml:"network-mode,omitempty"`
	// Ignite sandbox and kernel imageNames
	Sandbox string `yaml:"sandbox,omitempty"`
	Kernel  string `yaml:"kernel,omitempty"`
	// Override container runtime
	Runtime string `yaml:"runtime,omitempty"`
	// Set node CPU (cgroup or hypervisor)
	CPU float64 `yaml:"cpu,omitempty"`
	// Set node CPUs to use
	CPUSet string `yaml:"cpu-set,omitempty"`
	// Set node Memory (cgroup or hypervisor)
	Memory string `yaml:"memory,omitempty"`
	// Set the nodes Sysctl
	Sysctls map[string]string `yaml:"sysctls,omitempty"`
	// Extra options, may be kind specific
	Extras *Extras `yaml:"extras,omitempty"`
	// List of node names to wait for before satarting this particular node
	WaitFor []string `yaml:"wait-for,omitempty"`
}

// ConfigDispatcher represents the config of a configuration machine
// that is responsible to execute configuration commands on the nodes
// after they started.
type ConfigDispatcher struct {
	Vars map[string]interface{} `yaml:"vars,omitempty"`
}

type LinkConfig struct {
	Endpoints []string
	Labels    map[string]string      `yaml:"labels,omitempty"`
	Vars      map[string]interface{} `yaml:"vars,omitempty"`
}

// Extras contains extra node parameters which are not entitled to be part of a generic node config.
type Extras struct {
	SRLAgents []string `yaml:"srl-agents,omitempty"`
	// Nokia SR Linux agents. As of now just the agents spec files can be provided here
	MysocketProxy string `yaml:"mysocket-proxy,omitempty"`
	// Proxy address that mysocketctl will use
	CeosCopyToFlash []string `yaml:"ceos-copy-to-flash,omitempty"`
	// paths to files which are to be copied to ceos flash dir
}

// ref: https://github.com/srl-labs/containerlab/blob/v0.32.1/types/types.go#L359-L362
type LabData struct {
	Containers []ContainerDetails `json:"containers"`

	// In this definition, MySocketIo is missing because it isn't needed for us
}

// ref: https://github.com/srl-labs/containerlab/blob/v0.32.1/types/types.go#L318-L329
type ContainerDetails struct {
	LabName     string `json:"lab_name,omitempty"`
	LabPath     string `json:"labPath,omitempty"`
	Name        string `json:"name,omitempty"`
	ContainerID string `json:"container_id,omitempty"`
	Image       string `json:"image,omitempty"`
	Kind        string `json:"kind,omitempty"`

	State       string `json:"state,omitempty"`
	IPv4Address string `json:"ipv4_address,omitempty"`
	IPv6Address string `json:"ipv6_address,omitempty"`

	// In this definition, Group is missing because it doesn't seem to be used in `clab inspect`.
	// ref: https://github.com/srl-labs/containerlab/blob/v0.32.1/cmd/inspect.go#L137-L277
}
