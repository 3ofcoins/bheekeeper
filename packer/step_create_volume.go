package packer

import "fmt"
import "os/exec"
import "strconv"

import "github.com/mitchellh/multistep"
import "github.com/mitchellh/packer/packer"

type stepCreateVolume struct{}

func (s *stepCreateVolume) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Creating ZFS volume...")
	cmd := exec.Command("zfs", "create",
		"-V", strconv.FormatUint(uint64(config.VolumeSize), 10),
		config.VolumeName)
	// TODO: keep stderr

	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error creating ZFS volume: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	return multistep.ActionContinue
}

func (s *stepCreateVolume) Cleanup(state multistep.StateBag) {}
