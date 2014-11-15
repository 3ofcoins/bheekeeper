package cli

import cli "github.com/mitchellh/cli"

type Runner func([]string) int

type cmd struct {
	help, synopsis string
	run            Runner
}

func (c *cmd) Help() string {
	return c.help
}

func (c *cmd) Synopsis() string {
	return c.synopsis
}

func (c *cmd) Run(args []string) int {
	return c.run(args)
}

// "cli" is taken, no better idea
type CLI struct{ *cli.CLI }

func (c *CLI) cmdFactory(help, synopsis string, run Runner) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &cmd{help, synopsis, run}, nil
	}
}

func NewCLI(name, version string) *CLI {
	c := &CLI{CLI: cli.NewCLI(name, version)}
	c.Commands = make(map[string]cli.CommandFactory)
	return c
}

func (c *CLI) Register(cmd *Command) {
	cmd.RegisterInto(c.CLI)
}
