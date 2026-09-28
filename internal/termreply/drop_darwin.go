package termreply

import "golang.org/x/sys/unix"

// tcflush(fd, TCIFLUSH), as macOS's libc does it: TIOCFLUSH with FREAD.
func dropPending(fd int) error { return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, unix.TCIFLUSH) }
