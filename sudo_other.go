//go:build !android && !linux && !darwin && !windows
// +build !android,!linux,!darwin,!windows

package clicommon

import "errors"

func callSudo(action string, params []string) error {
	return errors.New("privilege escalation not supported on this platform")
}
