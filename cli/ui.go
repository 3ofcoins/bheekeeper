package cli

import "fmt"
import "os"

import "github.com/mgutz/ansi"

var DEBUG = true

var Colorize = struct {
	AsDebug, AsInfo, AsError func(string) string
}{
	ansi.ColorFunc("blue+h"),
	ansi.ColorFunc("yellow"),
	ansi.ColorFunc("red+h"),
}

func Debug(s string) {
	if DEBUG {
		fmt.Fprintln(os.Stderr, Colorize.AsDebug("DEBUG: "+s))
	}
}

func Debugf(format string, a ...interface{}) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, Colorize.AsDebug("DEBUG: "+format)+"\n", a...)
	}
}

func Output(s string) {
	fmt.Println(s)
}

func Printf(format string, a ...interface{}) {
	Output(fmt.Sprintf(format, a...))
}

func Info(s string) {
	fmt.Println(Colorize.AsInfo(s))
}

func Infof(format string, a ...interface{}) {
	Info(fmt.Sprintf(format, a...))
}

func Error(err error) {
	fmt.Fprintln(os.Stderr, Colorize.AsError("ERROR: "+err.Error()))
}

func Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, Colorize.AsError("ERROR: "+format)+"\n", a...)
}
