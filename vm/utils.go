package vm

import "bytes"
import "os"
import "os/exec"
import "strings"

func zfs(arg ...string) (out string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("zfs", arg...)
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	out = buf.String()
	return
}

func zfs_peek(cmd string, arg ...string) ([][]string, error) {
	arg = append([]string{cmd, "-H"}, arg...)
	if out, err := zfs(arg...); err != nil {
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
