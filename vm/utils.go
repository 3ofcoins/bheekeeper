package vm

import "bytes"
import "io"
import "os"
import "os/exec"
import "strings"
import "syscall"

import "github.com/3ofcoins/bheekeeper/cli"

var stderr = io.Writer(os.Stderr)

func withStderr(newStderr io.Writer, fn func()) {
	origStderr := stderr
	defer func() { stderr = origStderr }()
	stderr = newStderr
	fn()
}

func cmd(stdin io.Reader, stdout io.Writer, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func run(stdin io.Reader, stdout io.Writer, command string, args ...string) error {
	cli.Debugf("+ %s %v", command, args)
	return cmd(stdin, stdout, command, args...).Run()
}

func runStatus(stdin io.Reader, stdout io.Writer, command string, args ...string) (int, error) {
	if err := run(stdin, stdout, command, args...); err != nil {
		switch err.(type) {
		case *exec.ExitError:
			ws := err.(*exec.ExitError).Sys().(syscall.WaitStatus)
			if ws.Signaled() {
				cli.Debugf("%s killed by %s", command, ws.Signal())
				return ws.ExitStatus(), err
			} else {
				cli.Debugf("%s exited %d", command, ws.ExitStatus())
				return ws.ExitStatus(), nil
			}
		default:
			return -1, err
		}
	}
	return 0, nil
}

func runStdout(stdin io.Reader, command string, args ...string) (string, error) {
	var buf bytes.Buffer
	err := run(stdin, &buf, command, args...)
	return buf.String(), err
}

func zfs_peek(cmd string, arg ...string) ([][]string, error) {
	arg = append([]string{cmd, "-H"}, arg...)
	if out, err := runStdout(nil, "zfs", arg...); err != nil {
		return nil, err
	} else {
		lines := strings.Split(out, "\n")
		lines = lines[:len(lines)-1] // trailing newline leaves us with empty line at end
		words := make([][]string, len(lines))
		for ln, line := range lines {
			words[ln] = strings.Split(line, "\t")
		}
		return words, nil
	}
}
