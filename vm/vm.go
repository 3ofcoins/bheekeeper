package vm

import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "net"
import "os"
import "path/filepath"
import "strings"

var ErrVMNotFound = errors.New("VM not found")

type VM struct {
	Name, Volume string
	properties   map[string]string
	tap          string
}

func NewVM(name, volume string) *VM {
	return &VM{Name: name, Volume: volume}
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
				return vm, nil
			}
		}
	}
	return nil, ErrVMNotFound
}

func (vm *VM) Properties() map[string]string {
	if vm.properties == nil {
		if props, err := zfs_peek("get", "-o", "property,value", "all", vm.Volume); err != nil {
			panic(err) // no better idea here
		} else {
			vm.properties = make(map[string]string)
			for _, prop := range props {
				if !strings.HasPrefix(prop[0], "bhyve") {
					continue
				}
				vm.properties[prop[0][6:]] = prop[1]
			}
		}
	}
	return vm.properties
}

var PropertyDefaults = map[string]string{
	"memory":    "1024",
	"cpus":      "1",
	"grub:root": "hd0,msdos1",
}

func (vm *VM) Property(name string) string {
	if val, exists := vm.Properties()[name]; exists {
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

func (vm *VM) Tap() string {
	if vm.tap == "" {
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

func (vm *VM) Exists() bool {
	if _, err := os.Stat(filepath.Join("/dev/vmm", vm.Name)); err != nil {
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
}

func (vm *VM) volumePath() string {
	return filepath.Join("/dev/zvol", vm.Volume)
}

func (vm *VM) RunGrub() error {
	deviceMap, err := ioutil.TempFile("", "bheekeper_device.map_")
	if err != nil {
		return err
	}

	defer os.Remove(deviceMap.Name())

	if _, err := io.WriteString(deviceMap, fmt.Sprintf("(hd0) %s\n", vm.volumePath())); err != nil {
		return err
	}

	if err := deviceMap.Close(); err != nil {
		return err
	}

	return run(os.Stdin, os.Stdout, "grub-bhyve",
		"-r", vm.Property("grub:root"),
		"-m", deviceMap.Name(),
		"-M", vm.Property("mem"),
		vm.Name)
}

func (vm *VM) RunBhyve() error {
	defer vm.Destroy()
	return run(os.Stdin, os.Stdout, "bhyve",
		"-c", vm.Property("cpus"),
		"-m", vm.Property("mem"),
		"-A", "-P", "-H",
		"-s", "0,hostbridge",
		"-s", "1,lpc",
		"-s", "2,virtio-blk,"+vm.volumePath(),
		"-s", "3,virtio-net,"+vm.Tap(),
		"-l", "com1,stdio",
		vm.Name)
}
