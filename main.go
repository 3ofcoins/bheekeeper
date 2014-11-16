package main

import "fmt"
import "os"

import "github.com/3ofcoins/bheekeeper/cli"
import "github.com/3ofcoins/bheekeeper/vm"

var cmdList = cli.NewCommand("list", "List known VMs", func(args []string) error {
	if len(args) > 0 {
		return cli.ErrUsage
	}
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

var cmdStatus = newVMCommand("status", "Show detailed status of a VM", func(vm *vm.VM) error {
	cli.Printf("Name: %v\nExists: %v\nZFS Volume: %v\nProperties:",
		vm.Name, vm.Exists(), vm.Volume)
	for prop, val := range vm.Properties() {
		cli.Printf("  %v: %v", prop, val)
	}
	return nil
})

var cmdRun = newVMCommand("run", "Run VM", func(vm *vm.VM) error {
	//if err := vm.RunGrub(); err != nil {
	//	return err
	//}
	return vm.RunBhyve()
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
	c.Register(cmdList)
	c.Register(cmdStatus)
	c.Register(cmdRun)
	c.Register(cmdDestroy)

	exitStatus, err := c.Run()
	if err != nil {
		cli.Error(err)
	}
	os.Exit(exitStatus)
}
