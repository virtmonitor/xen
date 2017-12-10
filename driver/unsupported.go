// +build windows

package xen

import (
	"fmt"

	"github.com/virtmonitor/driver"
)

// Detect Detect dependencies
func (x *Xen) Detect() bool {
	return false
}

// Collect Collect domain statistics
func (x *Xen) Collect(cpu bool, block bool, network bool) (domains map[driver.DomainID]*driver.Domain, err error) {
	domains = make(map[driver.DomainID]*driver.Domain)
	err = fmt.Errorf("XEN not supported on this platform")
	return
}
