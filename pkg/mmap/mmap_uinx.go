//+build aix darwin dragonfly freebsd linux netbsd openbsd solaris, !windows

package mmap

import "golang.org/x/sys/unix"

func mapMMap(length int, inprot, inflags, fd uintptr, off int64) ([]byte, error) {
	flags := unix.MAP_SHARED
	prot := unix.PROT_READ
	switch {
	case inprot&COPY != 0:
		prot |= unix.PROT_WRITE
		flags = unix.MAP_PRIVATE
	case inprot&RDWR != 0:
		prot |= unix.PROT_WRITE
	case inprot&EXEC != 0:
		prot |= unix.PROT_EXEC
	case inflags&ANON != 0:
		flags |= unix.MAP_ANON
	}
	b, err := unix.Mmap(int(fd), off, length, prot, flags)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m MMap) flush() error {
	return unix.Msync([]byte(m), unix.MS_SYNC)
}

func (m MMap) lock() error {
	return unix.Mlock([]byte(m))
}

func (m MMap) unlock() error {
	return unix.Munlock([]byte(m))
}

func (m MMap) unmap() error {
	return unix.Munmap([]byte(m))
}
