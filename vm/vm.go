package vm

import "bytes"
import "crypto/md5"
import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "net"
import "os"
import "os/exec"
import "path/filepath"
import "regexp"
import "strconv"
import "strings"

import "github.com/3ofcoins/bheekeeper/cli" // FIXME? UI part seems awfully clunky

var ErrVMNotFound = errors.New("VM not found")

type VM struct {
	Name, Volume string
	Properties   map[string]string
	tap          string
	loaded       bool
	*exec.Cmd
}

func NewVM(name, volume string) *VM {
	return &VM{Name: name, Volume: volume, Properties: make(map[string]string)}
}

func AllVMs() ([]*VM, error) {
	if lines, err := zfs_peek("get", "-t", "volume", "-s", "local", "-o", "value,name", "bhyve:name"); err != nil {
		return nil, err
	} else {
		vms := make([]*VM, len(lines))
		for i, line := range lines {
			vms[i] = NewVM(line[0], line[1])
		}
		return vms, nil
	}
}

func FindVM(name string) (*VM, error) {
	if vms, err := AllVMs(); err != nil {
		return nil, err
	} else {
		for _, vm := range vms {
			if vm.Name == name {
				err := vm.LoadProperties()
				return vm, err
			}
		}
	}
	return nil, ErrVMNotFound
}

func (vm *VM) LoadProperties() error {
	props, err := zfs_peek("get", "-o", "property,value", "all", vm.Volume)
	if err != nil {
		return err
	}
	for _, prop := range props {
		if !strings.HasPrefix(prop[0], "bhyve:") {
			continue
		}
		vm.Properties[prop[0][6:]] = prop[1]
	}
	return nil
}

func (vm *VM) MAC() string {
	hsh := md5.Sum([]byte(vm.Name))
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0xAB
	hw[2] = 0xEE
	hw[3] = hsh[0]
	hw[4] = hsh[1]
	hw[5] = hsh[2]
	return hw.String()
}

var PropertyDefaults = map[string]string{
	"bridge":    "bridge0",
	"cpus":      "1",
	"grub:root": "hd0,msdos1",
	"mem":       "1024",
}

func (vm *VM) Property(name string) string {
	if val, exists := vm.Properties[name]; exists {
		return val
	} else {
		return PropertyDefaults[name]
	}
}

func (vm *VM) Bridge() string {
	bridge := vm.Property("bridge")
	if _, err := net.InterfaceByName(bridge); err != nil {
		if err := run(nil, os.Stdout, "ifconfig", bridge, "create"); err != nil {
			panic(err)
		}
	}
	return bridge
}

var rxSpace = regexp.MustCompile(`\s+`)

func (vm *VM) Tap(create bool) string {
	if pid := vm.BhyvePid(); vm.tap == "" && pid != 0 {
		if out, err := runStdout(nil, "fstat", "-p", strconv.Itoa(pid), "-f", "/dev"); err != nil {
			cli.Error(err)
		} else {
			for _, ln := range strings.Split(out, "\n") {
				if ln == "" {
					continue
				}
				lnw := rxSpace.Split(ln, -1)
				if dev := lnw[len(lnw)-2]; strings.HasPrefix(dev, "tap") {
					vm.tap = dev
				}
			}
		}
	}
	if vm.tap == "" && create {
		if tap, err := runStdout(nil, "ifconfig", "tap", "create"); err != nil {
			panic(err)
		} else {
			vm.tap = strings.TrimSpace(tap)
			if err := run(nil, os.Stdout, "ifconfig", vm.Bridge(), "addm", vm.tap); err != nil {
				panic(err)
			}
		}
	}
	return vm.tap
}

func (vm *VM) vmmPath() string {
	return filepath.Join("/dev/vmm", vm.Name)
}

func (vm *VM) BhyvePid() int {
	var out string
	var err error

	if !vm.Exists() {
		return 0
	}

	withStderr(nil, func() {
		out, err = runStdout(nil, "fuser", vm.vmmPath())
	})

	if err != nil {
		cli.Error(err)
		return 0
	}

	// Get just the first PID
	out = strings.Split(strings.TrimSpace(out), " ")[0]
	pid, _ := strconv.Atoi(out)
	return pid
}

func (vm *VM) Exists() bool {
	if _, err := os.Stat(vm.vmmPath()); err != nil {
		return !os.IsNotExist(err)
	} else {
		return true
	}
}

func (vm *VM) RunBhyvectl(args ...string) error {
	args = append([]string{"--vm=" + vm.Name}, args...)
	return run(nil, os.Stdout, "bhyvectl", args...)
}

func (vm *VM) Destroy() {
	if vm.Exists() {
		vm.RunBhyvectl("--destroy")
	}
	if vm.tap != "" {
		run(nil, os.Stdout, "ifconfig", vm.Bridge(), "deletem", vm.tap)
		run(nil, os.Stdout, "ifconfig", vm.tap, "destroy")
		vm.tap = ""
	}
	vm.loaded = false
}

func (vm *VM) volumePath() string {
	return filepath.Join("/dev/zvol", vm.Volume)
}

func (vm *VM) RunGrub(in io.Reader) error {
	deviceMap, err := ioutil.TempFile("", "bheekeper_device.map_")
	if err != nil {
		return err
	}

	defer os.Remove(deviceMap.Name())

	deviceMapLines := []string{fmt.Sprintf("(hd0) %s\n", vm.volumePath())}
	if iso := vm.Property("cdrom_iso"); iso != "" {
		deviceMapLines = append(deviceMapLines, fmt.Sprintf("(cd0) %s\n", iso))
	}

	if _, err := io.WriteString(deviceMap, strings.Join(deviceMapLines, "")); err != nil {
		return err
	}

	if err := deviceMap.Close(); err != nil {
		return err
	}

	return run(in, os.Stdout, "grub-bhyve",
		"-r", vm.Property("grub:root"),
		"-m", deviceMap.Name(),
		"-M", vm.Property("mem"),
		vm.Name)
}

var ErrLoaded = errors.New("Already loaded")

func (vm *VM) Load() error {
	if vm.loaded {
		return ErrLoaded
	}

	var grubInRd io.Reader
	if grubInStr, exists := vm.Properties["grub:in"]; exists {
		if grubInStr == "-" {
			grubInRd = os.Stdin
		} else if strings.HasPrefix(grubInStr, "\"") {
			grubInStr, err := strconv.Unquote(grubInStr)
			if err != nil {
				return err
			}
			grubInRd = bytes.NewBufferString(grubInStr)
		}
	}
	if err := vm.RunGrub(grubInRd); err != nil {
		return err
	}

	args := []string{
		"-c", vm.Property("cpus"),
		"-m", vm.Property("mem"),
		"-A", "-P", "-H",
		"-s", "0,hostbridge",
		"-s", "1,lpc",
		"-s", "2:0,virtio-blk," + vm.volumePath(),
		"-s", "3,virtio-net," + vm.Tap(true) + ",mac=" + vm.MAC(),
		"-l", "com1,stdio"}

	if iso := vm.Property("cdrom_iso"); iso != "" {
		args = append(args, "-s", "2:1,ahci-cd,"+iso)
	}

	args = append(args, vm.Name)

	vm.Cmd = exec.Command("bhyve", args...)
	vm.Stdin = os.Stdin
	vm.Stdout = os.Stdout
	vm.Stderr = os.Stderr

	vm.loaded = true
	return nil
}

func (vm *VM) EnsureLoaded() error {
	if !vm.loaded {
		return vm.Load()
	}
	return nil
}

func (vm *VM) Run() error {
	if err := vm.EnsureLoaded(); err != nil {
		return err
	}
	defer vm.Destroy()
	return vm.Run()
}
