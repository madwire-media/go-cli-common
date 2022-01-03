//go:build windows
// +build windows

package clicommon

import (
	"os"
	"syscall"

	"github.com/kardianos/osext"
	"golang.org/x/sys/windows"
)

func callSudo(action string, params []string) error {
	thisExe, err := osext.Executable()
	if err != nil {
		return err
	}

	// Inspired by https://stackoverflow.com/a/59147866/3052732
	verb := "runas"
	cwd, _ := os.Getwd()
	args := append([]string{sudoArg, action}, params...)

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(thisExe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(makeCmdLine(args))

	var showCmd int32 = 1 // SW_NORMAL

	return windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
}

// Copy of builtin Go algorithm: https://github.com/golang/go/blob/75032ad8cfac4aefbacd17b47346ac8c1b5ff33f/src/syscall/exec_windows.go#L44
// appendEscapeArg escapes the string s, as per escapeArg,
// appends the result to b, and returns the updated slice.
func appendEscapeArg(b []byte, s string) []byte {
	if len(s) == 0 {
		return append(b, `""`...)
	}

	needsBackslash := false
	hasSpace := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\':
			needsBackslash = true
		case ' ', '\t':
			hasSpace = true
		}
	}

	if !needsBackslash && !hasSpace {
		// No special handling required; normal case.
		return append(b, s...)
	}
	if !needsBackslash {
		// hasSpace is true, so we need to quote the string.
		b = append(b, '"')
		b = append(b, s...)
		return append(b, '"')
	}

	if hasSpace {
		b = append(b, '"')
	}
	slashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		default:
			slashes = 0
		case '\\':
			slashes++
		case '"':
			for ; slashes > 0; slashes-- {
				b = append(b, '\\')
			}
			b = append(b, '\\')
		}
		b = append(b, c)
	}
	if hasSpace {
		for ; slashes > 0; slashes-- {
			b = append(b, '\\')
		}
		b = append(b, '"')
	}

	return b
}

// Copy of builtin Go algorithm: https://github.com/golang/go/blob/75032ad8cfac4aefbacd17b47346ac8c1b5ff33f/src/syscall/exec_windows.go#L102
// makeCmdLine builds a command line out of args by escaping "special"
// characters and joining the arguments with spaces.
func makeCmdLine(args []string) string {
	var b []byte
	for _, v := range args {
		if len(b) > 0 {
			b = append(b, ' ')
		}
		b = appendEscapeArg(b, v)
	}
	return string(b)
}
