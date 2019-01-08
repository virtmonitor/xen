package xen

import "github.com/virtmonitor/driver"

const (
	//XLPath Path to XL
	XLPath = "/usr/sbin/xl"

	//XSPath Path to xenstore-ls
	XSPath = "/usr/bin/xenstore-ls"

	//BEPath Path to xen-backend devices
	BEPath = "/sys/bus/xen-backend/devices"

	//BETapName Tap name
	BETapName = "tap-%d-%s"

	//BEVbdName name
	BEVbdName = "vbd-%d-%s"

	//QmpSocketPath Path to the xen qmp sockets
	QmpSocketPath = "/var/run/xen"

	//QmpSocketName Name format for domain socket
	QmpSocketName = "qmp-libxenstat-%d"
)

const (
	//CPUOnline CPU is online
	CPUOnline = 1 << iota
	//CPURunning CPU is using cpu time
	CPURunning
	//CPUBlocked CPU is waiting for cpu time
	CPUBlocked
)

// Xen Xen object
type Xen struct {
	driver.Driver
}

//DomainID Domain ID
type DomainID driver.DomainID

// CPU CPU object
type CPU driver.CPU

// Block Block object
type Block driver.BlockDevice

// Network Network object
type Network driver.NetworkInterface

// Domain Domain object
type Domain driver.Domain

//Vcpu VCPU struct
type Vcpu struct {
	ID    driver.DomainID
	CPU   uint64
	State string
	Time  float64
}

//Name Driver name
func (x *Xen) Name() driver.DomainHypervisor {
	return driver.DomainHypervisor("XEN")
}

//Close Close Driver
func (x *Xen) Close() {}
