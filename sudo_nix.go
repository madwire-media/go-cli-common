// +build android linux darwin

package clicommon

import (
	"errors"
	"os"
	"os/exec"

	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"
)

func callSudo(action string, params []string) error {
	thisExe, err := osext.Executable()
	if err != nil {
		return err
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return errors.New("Not a terminal")
	}

	args := append([]string{thisExe, sudoArg, action}, params...)

	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
