//+build windows, !unix

package mmap

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"sync"
)

type addrinfo struct {
	file     windows.Handle
	mapview  windows.Handle
	writable bool
}

var handleLock sync.Mutex
var handleMap = map[uintptr]*addrinfo{}

func mapMMap(length int64, prot, flags, hfile uintptr, off int64) ([]byte, error) {
	flProtect := uint32(windows.PAGE_READONLY)
	dwDesiredAccess := uint32(windows.FILE_MAP_READ)
	writable := false
	switch {
	case prot&COPY != 0:
		flProtect = windows.PAGE_WRITECOPY
		dwDesiredAccess = windows.FILE_MAP_COPY
		writable = true
	case prot&RDWR != 0:
		flProtect = windows.PAGE_READWRITE
		dwDesiredAccess = windows.FILE_MAP_WRITE
		writable = true
	}
	if prot&EXEC != 0 {
		flProtect <<= 4
		dwDesiredAccess |= windows.FILE_MAP_EXECUTE
	}
	maxSizeHigh := uint32((off + int64(length)) >> 32)
	maxSizeLow := uint32((off + int64(length)) & 0xFFFFFFFF)
	h, errno := windows.CreateFileMapping(windows.Handle(hfile), nil, flProtect, maxSizeHigh, maxSizeLow, nil)
	if h == 0 {
		return nil, os.NewSyscallError("CreateFileMapping", errno)
	}
	fileOffsetHigh := uint32(off >> 32)
	fileOffsetLow := uint32(off & 0xFFFFFFFF)
	addr, errno := windows.MapViewOfFile(h, dwDesiredAccess, fileOffsetHigh, fileOffsetLow, uintptr(length))
	if addr == 0 {
		return nil, os.NewSyscallError("MapViewOfFile", errno)
	}
	handleLock.Lock()
	handleMap[addr] = &addrinfo{
		file:     windows.Handle(hfile),
		mapview:  h,
		writable: writable,
	}
	handleLock.Unlock()
	m := MMap{}
	dh := m.header()
	dh.Data = addr
	dh.Len = int(length)
	dh.Cap = dh.Len
	return m, nil
}

func (m MMap) flush() error {
	addr, length := m.addrLen()
	errno := windows.FlushViewOfFile(addr, length)
	if errno != nil {
		return os.NewSyscallError("FlushViewOfFile", errno)
	}
	handleLock.Lock()
	defer handleLock.Unlock()
	handle, ok := handleMap[addr]
	if !ok {
		// should be impossible
		return errors.New("Unknown base address")
	}
	if handle.writable {
		if err := windows.FlushFileBuffers(handle.file); err != nil {
			return os.NewSyscallError("FlushFileBuffers", err)
		}
	}
	return nil
}

func (m MMap) lock() error {
	addr, length := m.addrLen()
	errno := windows.VirtualLock(addr, length)
	return os.NewSyscallError("VirtualLock", errno)
}

func (m MMap) unlock() error {
	addr, length := m.addrLen()
	errno := windows.VirtualUnlock(addr, length)
	return os.NewSyscallError("VirtualUnlock", errno)
}

func (m MMap) unmap() error {
	err := m.flush()
	if err != nil {
		return err
	}
	addr := m.header().Data
	handleLock.Lock()
	defer handleLock.Unlock()
	err = windows.UnmapViewOfFile(addr)
	if err != nil {
		return err
	}
	handle, ok := handleMap[addr]
	if !ok {
		// should be impossible
		return errors.New("Unknown base address")
	}
	delete(handleMap, addr)
	e := windows.CloseHandle(handle.mapview)
	return os.NewSyscallError("CloseHandle", e)
}
