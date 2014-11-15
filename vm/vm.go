package vm

import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "os"
import "os/exec"
import "path/filepath"
import "strings"

var ErrVMNotFound = errors.New("VM not found")

type VM struct {
	Name, Volume string
	properties   map[string]string
}

func NewVM(name, volume string) *VM {
	return &VM{name, volume, nil}
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

func (vm *VM) Property(name, defval string) string {
	if val, exists := vm.Properties()[name]; exists {
		return val
	} else {
		return defval
	}
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
	cmd := exec.Command("bhyvectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var VMDefaultMemory = "1024"
var VMDefaultCpus = "1"
var VMDefaultGrubRoot = "hd0,msdos1"

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

	cmd := exec.Command("grub-bhyve",
		"-r", vm.Property("grub:root", VMDefaultGrubRoot),
		"-m", deviceMap.Name(),
		"-M", vm.Property("mem", VMDefaultMemory),
		vm.Name)

	// TODO: attach stdin only on demand
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (vm *VM) RunBhyve() error {
	cmd := exec.Command("bhyve",
		"-c", vm.Property("cpus", VMDefaultCpus),
		"-m", vm.Property("mem", VMDefaultMemory),
		"-A", "-P", "-H",
		"-s", "0,hostbridge",
		"-s", "1,lpc",
		"-s", "2,virtio-blk,"+vm.volumePath(),
		"-s", "3,virtio-net,tap0", // FIXME: create & destroy tap
		"-l", "com1,stdio",
		vm.Name)

	// TODO: attach stdin only on demand
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
