package model

type SystemInfo struct {
	Description string
	MAC         string
	IP          string
	Netmask     string
	Gateway     string
	Firmware    string
	Hardware    string
}

type ManagementIP struct {
	State   int
	VLAN    int
	MaxVLAN int
	IP      string
	Netmask string
	Gateway string
}

type Port struct {
	ID                int
	Enabled           bool
	TrunkMember       bool
	SpeedConfig       int
	SpeedActual       int
	FlowControlConfig int
	FlowControlActual int
}

type VLAN struct {
	ID            int
	Name          string
	TaggedPorts   []int
	UntaggedPorts []int
}

type VLANTable struct {
	Enabled  bool
	PortNum  int
	Count    int
	MaxVLANs int
	VLANs    []VLAN
}

type PortPVID struct {
	PortID int
	PVID   int
}
