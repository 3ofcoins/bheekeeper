package main

import "os"

import "github.com/3ofcoins/bheekeeper/cli"
import "github.com/3ofcoins/bheekeeper/vm"

func cmdList(args []string) int {
	if vms, err := vm.AllVMs(); err != nil {
		cli.Error(err)
		return 1
	} else {
		if len(vms) == 0 {
			cli.Info("No VMs configured")
		} else {
			cli.Info("Configured VMs:")
			for _, vm := range vms {
				if vm.Exists() {
					cli.Printf(" - %v (exists)", vm)
				} else {
					cli.Printf(" - %v", vm)
				}
			}
		}
		return 0
	}
}

func cmdStatus(args []string) int {
	vm, err := vm.FindVM(args[0])
	if err != nil {
		cli.Error(err)
		return 1
	}
	if vm == nil {
		cli.Errorf("VM not found: %v", args[0])
		return 1
	}
	header := vm.String()
	if vm.Exists() {
		header += " (exists)"
	}
	cli.Info(header)
	for prop, val := range vm.Properties() {
		cli.Printf("%v: %v", prop, val)
	}
	return 0
}

func cmdDestroy(args []string) int {
	vm, err := vm.FindVM(args[0])
	if err != nil {
		cli.Error(err)
		return 1
	}
	if vm == nil {
		cli.Errorf("VM not found: %v", args[0])
		return 1
	}
	if vm.Exists() {
		cli.Info("Destroying: " + vm.String())
		if err := vm.RunBhyvectl("--destroy"); err != nil {
			cli.Error(err)
			return 1
		} else {
			return 0
		}
	} else {
		cli.Errorf("VM does not exist: %s", vm.String())
		return 1
	}
}

func cmdRun(args []string) int {
	vm, err := vm.FindVM(args[0])
	if err != nil {
		cli.Error(err)
		return 1
	}
	if vm == nil {
		cli.Errorf("VM not found: %v", args[0])
		return 1
	}

	if err := vm.RunGrub(); err != nil {
		cli.Error(err)
		return 1
	}

	defer vm.RunBhyvectl("--destroy")

	if err := vm.RunBhyve(); err != nil {
		cli.Error(err)
		return 1
	}

	return 0
}

func main() {
	c := cli.NewCLI("bheekeeper", "0.0.1")
	c.Args = os.Args[1:]
	c.Cmd("list", "List VMs", "List VMs", cmdList)
	c.Cmd("status", "Show detailed status of VM", "Show VM status", cmdStatus)
	c.Cmd("run", "Run VM", "Run VM", cmdRun)
	c.Cmd("destroy", "Destroy VM", "Destroy VM", cmdDestroy)

	exitStatus, err := c.Run()
	if err != nil {
		cli.Error(err)
	}
	os.Exit(exitStatus)
}
