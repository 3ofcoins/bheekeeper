package cli

import "bytes"
import "errors"
import "flag"
import "fmt"
import "os"
import "strings"

import cli "github.com/mitchellh/cli"

var ErrUsage = errors.New("Incorrect usage")

type CommandFunc func([]string) error

type Command struct {
	name, usage, synopsis string
	runner                CommandFunc
	*flag.FlagSet
}

func NewCommand(usage, synopsis string, runner CommandFunc) *Command {
	name := strings.SplitN(usage, " ", 2)[0]
	cmd := &Command{name, usage, synopsis, runner, flag.NewFlagSet(name, flag.ContinueOnError)}
	cmd.FlagSet.Usage = func() { fmt.Fprintln(os.Stderr, cmd.Help()) }
	return cmd
}

func (cmd *Command) getDefaults() string {
	defer cmd.SetOutput(os.Stderr)
	buf := bytes.NewBuffer(nil)
	cmd.SetOutput(buf)
	cmd.PrintDefaults()
	return buf.String()
}

func (cmd *Command) Help() string {
	return fmt.Sprintf("Usage: %s\n\n%s\n\n%s",
		cmd.usage, cmd.synopsis, cmd.getDefaults())
}

func (cmd *Command) Synopsis() string {
	return cmd.synopsis
}

func (cmd *Command) Name() string {
	return cmd.name
}

func (cmd *Command) Usage() string {
	return cmd.usage
}

func (cmd *Command) Run(args []string) int {
	if err := cmd.Parse(args); err != nil {
		if err != flag.ErrHelp {
			Error(err)
		}
		return 1
	}
	if err := cmd.runner(cmd.Args()); err != nil {
		if err == ErrUsage {
			cmd.FlagSet.Usage()
			return 1
		}
		Error(err)
		return 2
	}
	return 0
}

func (cmd *Command) factory() (cli.Command, error) {
	return cmd, nil
}

func (cmd *Command) RegisterInto(c *cli.CLI) {
	if c.Commands == nil {
		c.Commands = make(map[string]cli.CommandFactory)
	}
	c.Commands[cmd.name] = cmd.factory
}
