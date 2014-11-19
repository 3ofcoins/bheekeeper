package main

import "fmt"
import "os"

import "github.com/3ofcoins/bheekeeper/cli"
import "github.com/3ofcoins/bheekeeper/vm"

var cmdStatus = cli.NewCommand("status [VM]", "List VMs or show detailed info about one",
	func(args []string) error {
		switch len(args) {
		case 0:
			if vms, err := vm.AllVMs(); err != nil {
				return err
			} else {
				if len(vms) == 0 {
					cli.Info("No VMs configured")
				} else {
					cli.Info("Configured VMs:")
					for _, vm := range vms {
						if vm.Exists() {
							cli.Printf(" *%v", vm.Name)
						} else {
							cli.Printf("  %v", vm.Name)
						}
					}
				}
			}
		case 1:
			if vm, err := vm.FindVM(args[0]); err != nil {
				return err
			} else {
				cli.Printf("Name: %v\nMAC: %s\nExists: %v\nZFS Volume: %v",
					vm.Name, vm.MAC(), vm.Exists(), vm.Volume)
				if vm.Exists() {
					if pid := vm.BhyvePid(); pid != 0 {
						cli.Printf("Bhyve PID: %d", pid)
					}
					if tap := vm.Tap(false); tap != "" {
						cli.Printf("Interface: %s", tap)
					}
				}
				cli.Output("Properties:")
				for prop, val := range vm.Properties {
					cli.Printf("  %v: %v", prop, val)
				}
			}
		default:
			return cli.ErrUsage
		}
		return nil
	})

func newVMCommand(name, synopsis string, runner func(*vm.VM) error) *cli.Command {
	return cli.NewCommand(name+" VM", synopsis, func(args []string) error {
		if len(args) != 1 {
			return cli.ErrUsage
		}
		if vm, err := vm.FindVM(args[0]); err != nil {
			return err
		} else {
			return runner(vm)
		}
	})
}

var cmdRun = newVMCommand("run", "Run VM", func(vm *vm.VM) error {
	return vm.Run()
})

var cmdDestroy = newVMCommand("destroy", "Destroy VM", func(vm *vm.VM) error {
	if vm.Exists() {
		cli.Info("Destroying: " + vm.Name)
		return vm.RunBhyvectl("--destroy")
	} else {
		return fmt.Errorf("VM does not exist: %s", vm.Name)
	}
})

func main() {
	c := cli.NewCLI("bheekeeper", "0.0.1")
	c.Args = os.Args[1:]
	c.Register(cmdStatus)
	c.Register(cmdRun)
	c.Register(cmdDestroy)

	exitStatus, err := c.Run()
	if err != nil {
		cli.Error(err)
	}
	os.Exit(exitStatus)
}
