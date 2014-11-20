package packer

import "fmt"
import "strings"

import "github.com/mitchellh/multistep"
import "github.com/mitchellh/packer/packer"
import "github.com/3ofcoins/bheekeeper/vm"

type bootCommandTemplateData struct {
	HTTPIP   string
	HTTPPort uint
	Name     string
}

type stepBoot struct {
	Tpl *packer.ConfigTemplate
	ch  chan error
}

func (s *stepBoot) Run(state multistep.StateBag) multistep.StepAction {
	s.ch = make(chan error, 1)
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	isoPath := state.Get("iso_path").(string)
	httpPort := state.Get("http_port").(uint)
	vm := config.vm

	tplData := &bootCommandTemplateData{
		config.HTTPIP,
		httpPort,
		vm.Name,
	}

	bootLines := make([]string, len(config.BootCommand))
	for i, cmd := range config.BootCommand {
		if cmd, err := s.Tpl.Process(cmd, tplData); err != nil {
			err := fmt.Errorf("Error preparing boot command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		} else {
			bootLines[i] = cmd
		}
	}

	config.vm.Properties["cdrom_iso"] = isoPath
	config.vm.Properties["grub:root"] = config.BootDevice
	config.vm.Properties["grub:in"] = strings.Join(bootLines, "")

	ui.Say("Loading machine...")
	if err := vm.Load(); err != nil {
		err := fmt.Errorf("Error running grub: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Booting...")
	go func() {
		s.ch <- vm.Run()
	}()
	return multistep.ActionContinue
}

func (s *stepBoot) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)
	vm := state.Get("vm").(*vm.VM)
	vm.Destroy()
	ui.Say(fmt.Sprintf("Terminated: %v", <-s.ch))
}
