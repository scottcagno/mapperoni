/*
 * // Copyright (c) 2021. Scott Cagno. All rights reserved.
 * // Use of this source code is governed by a BSD-style (clause 3)
 * // license that can be found in the root of this project in the LICENSE file.
 */

package mmap

import (
	"errors"
	"os"
	"reflect"
	"unsafe"
)

const (
	RDONLY = 0
	RDWR   = 1 << iota
	COPY
	EXEC
)

const (
	ANON = 1 << iota
)

type MMap []byte

func Map(f *os.File, prot, flags int) (MMap, error) {
	return MapRegion(f, -1, prot, flags, 0)
}

func MapRegion(f *os.File, length int, prot, flags int, offset int64) (MMap, error) {
	if offset%int64(os.Getpagesize()) != 0 {
		return nil, errors.New("Offset parameter must be a multiple of the system's pate size!")
	}
	var fd uintptr
	if flags&ANON == 0 {
		fd = f.Fd()
		if length < 0 {
			fi, err := f.Stat()
			if err != nil {
				return nil, err
			}
			length = int(fi.Size())
		}
	} else {
		if length <= 0 {
			return nil, errors.New("Anonymous mapping requires non-zero length!")
		}
		fd = ^uintptr(0)
	}
	return mapMMap(length, uintptr(prot), uintptr(flags), fd, offset)
}

func (m *MMap) header() *reflect.SliceHeader {
	return (*reflect.SliceHeader)(unsafe.Pointer(m))
}

func (m *MMap) addrLen() (uintptr, uintptr) {
	h := m.header()
	return h.Data, uintptr(h.Len)
}

func (m MMap) Lock() error {
	return m.lock()
}

func (m MMap) Unlock() error {
	return m.unlock()
}

func (m MMap) Flush() error {
	return m.flush()
}

func (m *MMap) Unmap() error {
	err := m.unmap()
	*m = nil
	return err
}
