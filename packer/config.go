package packer

import "errors"
import "fmt"
import "os/exec"
import "strings"

import "github.com/mitchellh/packer/common"
import "github.com/mitchellh/packer/packer"
import vboxcommon "github.com/mitchellh/packer/builder/virtualbox/common"

import "github.com/3ofcoins/bheekeeper/vm"

func zpool() string {
	cmd := exec.Command("zpool", "list", "-H")
	if out, err := cmd.Output(); err != nil {
		panic(err)
	} else {
		return strings.SplitN(string(out), "\t", 2)[0]
	}
}

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	vboxcommon.RunConfig      `mapstructure:",squash"`
	vboxcommon.ShutdownConfig `mapstructure:",squash"`
	vboxcommon.SSHConfig      `mapstructure:",squash"`

	VolumeSize      uint     `mapstructure:"volume_size"`
	VolumeName      string   `mapstructure:"volume_name"`
	ISOChecksum     string   `mapstructure:"iso_checksum"`
	ISOChecksumType string   `mapstructure:"iso_checksum_type"`
	ISOUrls         []string `mapstructure:"iso_urls"`
	VMName          string   `mapstructure:"vm_name"`
	BootCommand     []string `mapstructure:"boot_command"`
	BootDevice      string   `mapstructure:"boot_device"`

	RawSingleISOUrl string `mapstructure:"iso_url"`

	vm     *vm.VM
	HTTPIP string

	tpl *packer.ConfigTemplate
}

func NewConfig(raws ...interface{}) (*Config, []string, error) {
	var warns []string

	c := new(Config)
	md, err := common.DecodeConfig(c, raws...)
	if err != nil {
		return nil, nil, err
	}

	c.tpl, err = packer.NewConfigTemplate()
	if err != nil {
		return nil, nil, err
	}

	c.tpl.UserVars = c.PackerUserVars

	errs := common.CheckUnusedConfig(md)
	errs = packer.MultiErrorAppend(errs, c.RunConfig.Prepare(c.tpl)...)
	errs = packer.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(c.tpl)...)
	errs = packer.MultiErrorAppend(errs, c.SSHConfig.Prepare(c.tpl)...)

	if c.VolumeSize == 0 {
		c.VolumeSize = 40000
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s", c.PackerBuildName)
	}

	if c.VolumeName == "" {
		c.VolumeName = fmt.Sprintf("%s/%s", zpool(), c.VMName)
		warns = append(warns, fmt.Sprintf("volume_name not provided, using %s", c.VolumeName))
	}

	if c.BootDevice == "" {
		c.BootDevice = "(cd0)"
	}

	templates := map[string]*string{
		"iso_checksum":      &c.ISOChecksum,
		"iso_checksum_type": &c.ISOChecksumType,
		"iso_url":           &c.RawSingleISOUrl,
		"vm_name":           &c.VMName,
	}

	for n, ptr := range templates {
		var err error
		*ptr, err = c.tpl.Process(*ptr, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error processing %s: %s", n, err))
		}
	}

	for i, url := range c.ISOUrls {
		var err error
		c.ISOUrls[i], err = c.tpl.Process(url, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error processing iso_urls[%d]: %s", i, err))
		}
	}

	for i, command := range c.BootCommand {
		if err := c.tpl.Validate(command); err != nil {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("Error processing boot_command[%d]: %s", i, err))
		}
	}

	if c.ISOChecksumType == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("The iso_checksum_type must be specified."))
	} else {
		c.ISOChecksumType = strings.ToLower(c.ISOChecksumType)
		if c.ISOChecksumType != "none" {
			if c.ISOChecksum == "" {
				errs = packer.MultiErrorAppend(
					errs, errors.New("Due to large file sizes, an iso_checksum is required"))
			} else {
				c.ISOChecksum = strings.ToLower(c.ISOChecksum)
			}

			if h := common.HashForType(c.ISOChecksumType); h == nil {
				errs = packer.MultiErrorAppend(
					errs,
					fmt.Errorf("Unsupported checksum type: %s", c.ISOChecksumType))
			}
		}
	}

	if c.RawSingleISOUrl == "" && len(c.ISOUrls) == 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("One of iso_url or iso_urls must be specified."))
	} else if c.RawSingleISOUrl != "" && len(c.ISOUrls) > 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("Only one of iso_url or iso_urls may be specified."))
	} else if c.RawSingleISOUrl != "" {
		c.ISOUrls = []string{c.RawSingleISOUrl}
	}

	for i, url := range c.ISOUrls {
		c.ISOUrls[i], err = common.DownloadableURL(url)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Failed to parse iso_url %d: %s", i+1, err))
		}
	}

	if c.ISOChecksumType == "none" {
		warns = append(warns,
			"A checksum type of 'none' was specified. Since ISO files are so big,\n"+
				"a checksum is highly recommended.")
	}

	if c.ShutdownCommand == "" {
		warns = append(warns,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	// VM stuff
	c.vm = vm.NewVM(c.VMName, filepath.Join("/dev/zvol", c.VolumeName))

	// Bridge address
	if iface, err := net.InterfaceByName(c.vm.Bridge()); err != nil {
		errs = packer.MultiErrorAppend(errs, err)
	} else {
		if addrs, err := iface.Addrs(); err != nil {
			errs = packer.MultiErrorAppend(errs, err)
		} else {
			if len(addrs) == 0 {
				errs := packer.MultiErrorAppend(errs, fmt.Errorf("No address found for %s", iface))
			} else {
				c.HTTPIP = strings.SplitN(addrs[0].String(), "/", 2)[0]
			}
		}
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, warns, errs
	}

	return c, warns, nil
}
