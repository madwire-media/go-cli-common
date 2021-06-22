package clicommon

import (
	"errors"
	"fmt"
	"os"

	"github.com/kardianos/osext"
)

const sudoArg = "__sudo"

// CallSudo asks the user for superuser permissions, and then executes the
// currently-running program with those permissions for a particular action.
// TryHandleSudo should be called at the beginning of the program's main()
// function to catch these sudo calls.
func CallSudo(action string, params ...string) error {
	return callSudo(action, params)
}

// TryHandleSudo catches superuser self-executions to do certain actions that
// require superuser permissions
func TryHandleSudo() {
	if len(os.Args) >= 3 && os.Args[1] == sudoArg {
		action := os.Args[2]
		params := os.Args[3:]

		err := handleSudo(action, params)
		if err != nil {
			fmt.Println("Error handling sudo action")
			fmt.Println(err)
			os.Exit(1)
		}

		os.Exit(0)
	}
}

func handleSudo(action string, params []string) error {
	switch action {
	case "replaceExecutable":
		if len(params) < 1 {
			return errors.New("Not enough parameters for replaceExecutable action")
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

	default:
		return errors.New("Unknown sudo action")
	}

	return nil
}
