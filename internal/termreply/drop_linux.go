package termreply

import "golang.org/x/sys/unix"

// tcflush(fd, TCIFLUSH), as glibc does it: TCFLSH with the queue by value.
func dropPending(fd int) error { return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH) }
