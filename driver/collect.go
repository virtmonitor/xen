// +build !windows

package xen

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/digitalocean/go-qemu/qemu"
	"github.com/digitalocean/go-qemu/qmp"
	"github.com/virtmonitor/driver"
	"github.com/virtmonitor/virNetTap"
)

// Collect Collect domain statistics
func (x *Xen) Collect(cpu bool, block bool, network bool) (domains map[driver.DomainID]*driver.Domain, err error) {

	//log.Print("Collecting...")

	var procnet virNetTap.VirNetTap

	domains = make(map[driver.DomainID]*driver.Domain)
	nstat := make(map[string]virNetTap.InterfaceStats)
	xspaths := make(map[string]string)

	var domainIds []driver.DomainID
	var path string

	//log.Print("XS")

	if xspaths, domainIds, err = XenstoreLS(""); err != nil {
		return
	}

	// Dom0 is not counted
	if len(domainIds) < 1 {
		return
	}

	if nstat, err = procnet.GetAllVifStats(); err != nil {
		return
	}

	//log.Print("VIF")

	Paths := xspaths

	//log.Print("LOOP")

	// skip Domain0 -- i == domain index (id)
	for _, i := range domainIds {

		domain := &driver.Domain{}

		// get name
		name := Paths[fmt.Sprintf("/local/domain/%d/name", i)]

		// get uuid
		vmpath := Paths[fmt.Sprintf("/local/domain/%d/vm", i)]
		uuid := strings.Split(vmpath, "/")[2]

		domain.ID = driver.DomainID(i)
		domain.Name = name
		domain.UUID = uuid

		// get ostype
		ostype := Paths[fmt.Sprintf("/vm/%s/image/ostype", uuid)]
		domain.OSType = ostype

		if block {
			//log.Print("BLOCK")
			if domain.Blocks, err = BlockStats(domain, xspaths); err != nil {
				return
			}
		}

		if network {
			//log.Print("Network")
			dnetworks := []driver.NetworkInterface{}
			//log.Print("Network LOOP")

			for key := range Paths {
				fields := strings.Split(key, "/")
				if len(fields) == 6 && fields[1] == "libxl" && fields[2] == strconv.Itoa(int(i)) && fields[3] == "device" && fields[4] == "vif" {
					n, _ := strconv.ParseUint(fields[5], 10, 64)
					path = fmt.Sprintf("/libxl/%d/device/vif/%d", i, n)

					//log.Print("XS VIF")

					dnetwork := driver.NetworkInterface{}

					//viftype := Paths[fmt.Sprintf("%s/%s", path, "type")]
					dnetwork.Bridge = Paths[fmt.Sprintf("%s/%s", path, "bridge")]

					dnetwork.Mac, _ = net.ParseMAC(Paths[fmt.Sprintf("%s/%s", path, "mac")])

					// detect vif name and grab network statistics

					vifstat := virNetTap.InterfaceStats{}
					var ok bool

					if vifstat, ok = nstat[fmt.Sprintf("vif%d.%d-emu", i, n)]; ok {
						//vif1.0-emu
						dnetwork.Name = vifstat.VIF
					} else if vifstat, ok = nstat[fmt.Sprintf("vif%d.%d", i, n)]; ok {
						//vif1.0
						dnetwork.Name = vifstat.VIF
					} else if vifstat, ok = nstat[fmt.Sprintf("vif%s.%d", name, n)]; ok {
						//vif<name>.0
						dnetwork.Name = vifstat.VIF
					}

					// we may not have detected vif name by now
					if dnetwork.Name != "" {
						dnetwork.RX = driver.NetworkIO{Bytes: vifstat.IN.Bytes, Packets: vifstat.IN.Pkts, Errors: vifstat.IN.Errs, Drops: vifstat.IN.Drops}
						dnetwork.TX = driver.NetworkIO{Bytes: vifstat.OUT.Bytes, Packets: vifstat.OUT.Pkts, Errors: vifstat.OUT.Errs, Drops: vifstat.OUT.Drops}
					}

					dnetworks = append(dnetworks, dnetwork)

				}

			}

			domain.Interfaces = dnetworks
		}

		domains[domain.ID] = domain

	}

	//log.Print("CPU")

	if cpu {

		vcpulist := make(map[driver.DomainID][]driver.CPU)

		var id driver.DomainID
		var dcpus []driver.CPU

		if vcpulist, err = x.VcpuList(domainIds); err != nil {
			return
		}

		for id, dcpus = range vcpulist {
			domains[id].Cpus = dcpus
		}

	} // end of cpu

	//log.Print("COLLECT RETURN")

	return
}

// VcpuList xl vcpu-list ...
func (x *Xen) VcpuList(domid []driver.DomainID) (dcpus map[driver.DomainID][]driver.CPU, err error) {

	var id driver.DomainID
	var cmd *exec.Cmd
	var cmdOut io.ReadCloser

	dcpus = make(map[driver.DomainID][]driver.CPU)

	// vcpu-list does not require a list of domain id's, so an empty []Domid can be supplied

	if len(domid) <= 0 {
		cmd = exec.Command(XLPath, "vcpu-list")
	} else {
		args := []string{"vcpu-list"}
		for _, id = range domid {
			args = append(args, string(strconv.Itoa(int(id))))
		}
		cmd = exec.Command(XLPath, args...)
	}

	if cmdOut, err = cmd.StdoutPipe(); err != nil {
		return
	}

	var scanner *bufio.Scanner
	scanner = bufio.NewScanner(cmdOut)

	var vcpu uint64
	var cpu uint64
	var state string
	var time float64
	var dummy string
	var i int

	if err = cmd.Start(); err != nil {
		return
	}

	for scanner.Scan() {

		var dcpu driver.CPU

		i, err = fmt.Sscanf(scanner.Text(),
			"%s %d %d %d %s %g %s",
			&dummy, &id, &vcpu, &cpu, &state, &time, &dummy)

		if i < 7 {
			continue
		}

		if err != nil {
			return
		}

		if state == "r--" {
			dcpu.Flags |= driver.CPUOnline | driver.CPURunning
		} else if state == "-b-" {
			dcpu.Flags |= driver.CPUOnline | driver.CPUHalted
		} else if state == "--p" {
			dcpu.Flags |= driver.CPUPaused
		}

		dcpu.ID = vcpu
		dcpu.Time = time

		dcpus[id] = append(dcpus[id], dcpu)

	}

	if err = scanner.Err(); err != nil {
		return
	}

	if err = cmd.Wait(); err != nil {
		err = fmt.Errorf("Vcpu Wait error: %s (%+v)", err, cmd)
		return
	}

	return
}

// BlockStats IO Stats for this block device
func BlockStats(domain *driver.Domain, xspath map[string]string) (blocks []driver.BlockDevice, err error) {

	switch strings.ToUpper(domain.OSType) {
	case "HVM":
		return BlockStatsHVM(domain, xspath)
	default:
		return BlockStatsVBD(domain, xspath)
	}

}

// BlockStatsHVM IO Stats for HVM domain
func BlockStatsHVM(domain *driver.Domain, xspath map[string]string) (blocks []driver.BlockDevice, err error) {

	var path string
	path = filepath.Join(QmpSocketPath, fmt.Sprintf(QmpSocketName, domain.ID))

	if _, err = os.Stat(path); err == nil {

		var qsocket *qmp.SocketMonitor
		var qdomain *qemu.Domain

		var qstats []qemu.BlockStats
		var qstat qemu.BlockStats

		if qsocket, err = qmp.NewSocketMonitor("unix", path, 2*time.Second); err != nil {
			return
		}

		if err = qsocket.Connect(); err != nil {
			return
		}

		if qdomain, err = qemu.NewDomain(qsocket, string(domain.ID)); err != nil {
			return
		}
		defer qdomain.Close()

		if qstats, err = qdomain.BlockStats(); err != nil {
			return
		}

		var sname []string
		var qname string
		var ok bool
		var value string
		//var spath []string

		for _, qstat = range qstats {

			var dblock driver.BlockDevice

			qname = qstat.Device

			sname = strings.Split(qstat.Device, "-")
			if len(sname) == 2 {
				qname = sname[1]
			}

			dblock.Name = qname

			if govalidator.IsInt(qname) {
				// check /libxl/<domid>/device/vbd/<qname>/device-type
				if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/device-type", domain.ID, qname)]; ok {
					if value == "cdrom" {
						dblock.IsCDrom = true
					} else if value == "disk" {
						dblock.IsDisk = true
					}
				}
				// get mode
				if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/mode", domain.ID, qname)]; ok {
					if value == "r" {
						dblock.ReadOnly = true
					}
				}
			} else {
				// check vbd devices for matching device name and grab vdevs
				// /libxl/<domid>/device/vbd/<vdev>/<?>
				re := regexp.MustCompile(fmt.Sprintf("^/libxl/%d/device/vbd/[0-9]{1,}/dev", domain.ID))
				for path, value = range xspath {
					if re.FindString(path) != "" {
						if value == qname {
							// we found the matching vdev
							vdev := strings.Split(path, "/")[5]
							if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/device-type", domain.ID, vdev)]; ok {
								if value == "cdrom" {
									dblock.IsCDrom = true
								} else if value == "disk" {
									dblock.IsDisk = true
								}
							}
							if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/mode", domain.ID, vdev)]; ok {
								if value == "r" {
									dblock.ReadOnly = true
								}
							}
							break
						}
					}
				}
			}

			dblock.Read = driver.BlockIO{Operations: uint64(qstat.ReadOperations), Bytes: qstat.ReadBytes, Sectors: 0}
			dblock.Write = driver.BlockIO{Operations: uint64(qstat.WriteOperations), Bytes: qstat.WriteBytes, Sectors: 0}
			dblock.Flush = driver.BlockIO{Operations: uint64(qstat.FlushOperations), Bytes: 0, Sectors: 0}

			blocks = append(blocks, dblock)

		}

	}

	return

}

// BlockStatsVBD IO Stats for PV domain
func BlockStatsVBD(domain *driver.Domain, xspath map[string]string) (blocks []driver.BlockDevice, err error) {

	var path string
	var matches []string

	path = filepath.Join(BEPath, fmt.Sprintf(BETapName, domain.ID, "*"))

	if matches, err = filepath.Glob(path); err == nil {
		if len(matches) > 0 {
			// PV Tap device
			goto read
		}
	}

	path = filepath.Join(BEPath, fmt.Sprintf(BEVbdName, domain.ID, "*"))

	if matches, err = filepath.Glob(path); err == nil {
		if len(matches) > 0 {
			// PV VBD device
			goto read
		}
	}

	err = fmt.Errorf("Could not determine domain block device backends")
	return

read:

	var dir string
	var sdir string
	var files []os.FileInfo
	var file os.FileInfo
	var data []byte
	var str string
	var val uint64

	var vdev string
	var value string
	var ok bool

	for _, dir = range matches {
		if file, err = os.Stat(dir); err != nil {
			return
		}

		if !file.IsDir() {
			continue
		}

		sdir = filepath.Join(dir, "statistics")

		if files, err = ioutil.ReadDir(sdir); err != nil {
			return
		}

		var dblock driver.BlockDevice

		// get vdev/name from dir path
		// /sys/bus/xen-backend/devices/<driver>-<domid>-<vdev>
		vdev = strings.Split(strings.Split(dir, "/")[5], "-")[2]
		if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/dev", domain.ID, vdev)]; ok {
			dblock.Name = value
		}

		// get vdev type
		if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/device-type", domain.ID, vdev)]; ok {
			if value == "cdrom" {
				dblock.IsCDrom = true
			} else if value == "disk" {
				dblock.IsDisk = true
			}
		}

		// get mode
		if value, ok = xspath[fmt.Sprintf("/libxl/%d/device/vbd/%s/mode", domain.ID, vdev)]; ok {
			if value == "r" {
				dblock.ReadOnly = true
			}
		}

		for _, file = range files {

			if file.Name() == "ds_sect" || file.Name() == "oo_sect" {
				continue
			}

			if data, err = ioutil.ReadFile(filepath.Join(sdir, file.Name())); err != nil {
				return
			}

			str = strings.TrimSpace(string(data))
			val, _ = strconv.ParseUint(str, 10, 64)

			switch strings.ToLower(file.Name()) {
			case "rd_req":
				dblock.Read.Operations = val
			case "rd_sect":
				dblock.Read.Bytes = val * 512
			case "wr_req":
				dblock.Write.Operations = val
			case "wr_sect":
				dblock.Write.Bytes = val * 512
			//case "ds_sect":
			//	continue
			case "f_req":
				dblock.Flush.Operations = val
			//case "oo_sect":
			//	continue
			default:
			}

		}

		blocks = append(blocks, dblock)

	}

	return

}
