// +build !windows

package xen

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Detect Detect dependencies
func (x *Xen) Detect() bool {
	var err error

	var f *os.File
	if f, err = os.Open("/proc/xen/capabilities"); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Printf("Error detecting XEN driver: %v", err)
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "control_d") {
			goto detect_xl
		}
	}

	return false

detect_xl:

	if _, err = exec.LookPath(XLPath); err == nil {
		goto detect_xs
	}

	return false

detect_xs:

	if _, err = exec.LookPath(XSPath); err == nil {
		return true
	}

	return false

}
