//go:build !linux || !cgo

package webapi

import "errors"

func authenticateSystemUser(_, _ string) error {
	return errors.New("Linux PAM authentication requires a Linux CGO build")
}
