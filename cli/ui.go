package cli

import "fmt"
import "os"

import "github.com/mitchellh/cli"

var Ui = &cli.ColoredUi{
	// FIXME: a nicer approach would not hurt
	OutputColor: cli.UiColorNone,
	InfoColor:   cli.UiColorYellow,
	ErrorColor:  cli.UiColorRed,
	Ui: &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	},
}

func Output(s string) {
	Ui.Output(s)
}

func Printf(format string, a ...interface{}) {
	Output(fmt.Sprintf(format, a...))
}

func Info(s string) {
	Ui.Info(s)
}

func Infof(format string, a ...interface{}) {
	Info(fmt.Sprintf(format, a...))
}

func Error(err error) {
	Ui.Error(err.Error())
}

func Errorf(format string, a ...interface{}) {
	Ui.Error(fmt.Sprintf(format, a...))
}
