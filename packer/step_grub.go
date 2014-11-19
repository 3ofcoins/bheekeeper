package packer

import "fmt"
import "io"
import "time"

import "github.com/mitchellh/multistep"
import "github.com/mitchellh/packer/packer"
import "github.com/3ofcoins/bheekeeper/vm"

type bootCommandTemplateData struct {
	HTTPIP   string
	HTTPPort uint
	Name     string
}

type stepGrub struct {
	Tpl *packer.ConfigTemplate
}

func (s *stepGrub) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	isoPath := state.Get("iso_path").(string)
	httpPort := state.Get("http_port").(uint)

	vm := config.vm
	c.vm.Properties()["cdrom_iso"] = isoPath
	c.vm.Properties()["grub:root"] = config.BootDevice

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

	rd, wr := io.Pipe()
	go func() {
		time.Sleep(1 * time.Second)
		for _, line := range bootLines {
			// TODO: explicit <wait> <wait5> <wait10>
			wr.Write([]byte(line))
			time.Sleep(1 * time.Second)
		}
		wr.Close()
	}()

	ui.Say("Running Grub...")
	if err := vm.RunGrub(rd); err != nil {
		err := fmt.Errorf("Error running grub: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	return multistep.ActionContinue
}

func (s *stepGrub) Cleanup(state multistep.StateBag) {
	vm := state.Get("vm").(*vm.VM)
	vm.Destroy()
}
