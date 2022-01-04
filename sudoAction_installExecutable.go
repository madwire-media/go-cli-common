package clicommon

import (
	"errors"
	"os"
)

func init() {
	RegisterAction(InstallExecutableSudoAction{})
}

type InstallExecutableSudoAction struct {
	SourcePath      string
	DestinationPath string
}

func (a InstallExecutableSudoAction) Name() string {
	return "installExecutable"
}

func (a InstallExecutableSudoAction) Params() []string {
	return []string{a.SourcePath, a.DestinationPath}
}

func (a InstallExecutableSudoAction) Handle(params []string) error {
	if len(params) < 2 {
		return errors.New("not enough parameters for InstallExecutableSudoAction")
	}

	sourcePath := params[0]
	destinationPath := params[1]

	if sourcePath == "" {
		return errors.New("missing value for SourcePath")
	}
	if destinationPath == "" {
		return errors.New("missing value for DestinationPath")
	}

	err := os.Rename(sourcePath, destinationPath)
	if err != nil {
		return err
	}

	err = os.Chmod(destinationPath, 0755)
	if err != nil {
		return err
	}

	return nil
}
