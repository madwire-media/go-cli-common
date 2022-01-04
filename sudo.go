package clicommon

import (
	"errors"
	"fmt"
	"os"
)

const sudoArg = "__sudo"

type SudoAction interface {
	Name() string
	Params() []string
	Handle(params []string) error
}

var registeredActions = map[string]SudoAction{}

func RegisterAction(action SudoAction) {
	registeredActions[action.Name()] = action
}

// CallSudo asks the user for superuser permissions, and then executes the
// currently-running program with those permissions for a particular action.
// TryHandleSudo should be called at the beginning of the program's main()
// function to catch these sudo calls.
func CallSudo(action SudoAction) error {
	return callSudo(action.Name(), action.Params())
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
	if handler, ok := registeredActions[action]; ok {
		return handler.Handle(params)
	}

	return errors.New("unknown sudo action")
}
