// +build !windows

package xen

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/virtmonitor/driver"
)

// XenstoreLS xenstore-ls -f
func XenstoreLS(path string) (Paths map[string]string, domainIds []driver.DomainID, err error) {

	var cmd *exec.Cmd
	var cmdOut io.ReadCloser

	Paths = make(map[string]string)

	if path == "" {
		cmd = exec.Command(XSPath, "-f")
	} else {
		cmd = exec.Command(XSPath, "-f", path)
	}

	if cmdOut, err = cmd.StdoutPipe(); err != nil {
		return
	}

	var scanner *bufio.Scanner
	scanner = bufio.NewScanner(cmdOut)

	var xspath string
	var value string
	var i int

	if err = cmd.Start(); err != nil {
		return
	}

	for scanner.Scan() {
		i, err = fmt.Sscanf(scanner.Text(), "%s = %s", &xspath, &value)
		if i < 2 || err != nil {
			continue
		}

		Paths[xspath] = trimValue(value)
	}

	if err = scanner.Err(); err != nil {
		return
	}

	if err = cmd.Wait(); err != nil {
		return
	}

	for key, value := range Paths {
		fields := strings.Split(key, "/")
		if fields[len(fields)-1] == "domid" {
			id, _ := strconv.ParseUint(value, 10, 64)
			if int(id) == 0 {
				continue
			}
			domainIds = append(domainIds, driver.DomainID(id))
		}
	}

	return
}

func trimValue(v string) string {
	if len(v) > 0 && v[0] == '"' {
		v = v[1:]
	}
	if len(v) > 0 && v[len(v)-1] == '"' {
		v = v[:len(v)-1]
	}
	return v
}

// XSPathExists Check if path exists in map
func XSPathExists(Paths map[string]string, path string) (ok bool) {
	_, ok = Paths[path]
	return
}

// XSGetValue Returns value of path
func XSGetValue(Paths map[string]string, path string) (value string, ok bool) {
	value, ok = Paths[path]
	return
}
