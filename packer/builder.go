package packer

import "fmt"
import "os"
import "path/filepath"

// import "time"

import "github.com/mitchellh/multistep"
import "github.com/mitchellh/packer/common"
import "github.com/mitchellh/packer/packer"
import vboxcommon "github.com/mitchellh/packer/builder/virtualbox/common"

const BuilderId = "3ofcoins.bheekeeper"

var logf *os.File

func init() {
	if f, err := os.Create("/tmp/pbbk.log"); err != nil {
		panic(err)
	} else {
		logf = f
	}
	logln("HAHA INIT")
}

func logln(args ...interface{}) {
	fmt.Fprintln(logf, args...)
}

type Builder struct {
	config *Config
	runner multistep.Runner
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	logln("HAHA PREPARE")

	c, warnings, errs := NewConfig(raws...)
	if errs != nil {
		return warnings, errs
	}
	b.config = c

	return warnings, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	logln("HAHA RUN 1")

	steps := []multistep.Step{
		&common.StepDownload{
			Checksum:     b.config.ISOChecksum,
			ChecksumType: b.config.ISOChecksumType,
			Description:  "ISO",
			ResultKey:    "iso_path",
			Url:          b.config.ISOUrls,
		},
		&stepCreateVolume{},
		&vboxcommon.StepHTTPServer{
			HTTPDir:     b.config.HTTPDir,
			HTTPPortMin: b.config.HTTPPortMin,
			HTTPPortMax: b.config.HTTPPortMax,
		},
		&stepGrub{
			Tpl: b.config.tpl,
		},
		// &common.StepConnectSSH{
		// 	SSHAddress:     SSHAddress(b.config.Host, b.config.Port),
		// 	SSHConfig:      SSHConfig(b.config.SSHUsername, b.config.SSHPassword, b.config.SSHPrivateKeyFile),
		// 	SSHWaitTimeout: 1 * time.Minute,
		// },
		// &common.StepProvision{},
	}

	// Setup the state bag and initial state for the steps
	state := new(multistep.BasicStateBag)
	state.Put("cache", cache)
	state.Put("config", b.config)
	state.Put("hook", hook)
	state.Put("ui", ui)

	logln("HAHA RUN 2")

	// Run!
	if b.config.PackerDebug {
		b.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		b.runner = &multistep.BasicRunner{Steps: steps}
	}

	logln("HAHA RUN 3")

	b.runner.Run(state)

	logln("HAHA RUN 4")

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	logln("HAHA RUN 5")

	// No errors, must've worked
	artifact := &NullArtifact{}
	return artifact, nil
}

func (b *Builder) Cancel() {
	logln("HAHA CANCEL")
	if b.runner != nil {
		logln("Cancelling the step runner...")
		b.runner.Cancel()
	}
}
