package clicommon

import (
	"errors"
	"os"

	"github.com/kardianos/osext"
)

func init() {
	RegisterAction(ReplaceExecutableSudoAction{})
}

type ReplaceExecutableSudoAction struct {
	NewExe string
}

func (a ReplaceExecutableSudoAction) Name() string {
	return "replaceExecutable"
}

func (a ReplaceExecutableSudoAction) Params() []string {
	return []string{a.NewExe}
}

func (a ReplaceExecutableSudoAction) Handle(params []string) error {
	if len(params) < 1 {
		return errors.New("not enough parameters for ReplaceExecutableSudoAction")
	}

	newExe := params[0]

	thisExe, err := osext.Executable()
	if err != nil {
		return err
	}

	err = os.Rename(newExe, thisExe)
	if err != nil {
		return err
	}

	return nil
}
