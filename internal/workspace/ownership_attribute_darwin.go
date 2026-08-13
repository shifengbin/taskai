//go:build darwin

package workspace

import (
	"errors"

	"golang.org/x/sys/unix"
)

func isMissingOwnershipAttribute(err error) bool {
	return errors.Is(err, unix.ENOATTR)
}
