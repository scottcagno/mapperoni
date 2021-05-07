/*
 * // Copyright (c) 2021. Scott Cagno. All rights reserved.
 * // Use of this source code is governed by a BSD-style (clause 3)
 * // license that can be found in the root of this project in the LICENSE file.
 */

package mmap

import (
	"log"
	"os"
	"path/filepath"
)

const (
	KB    int  = 1 << 10
	MB    int  = 1 << 20
	PAGE  int  = 4 * KB // page size
	EMPTY byte = 0xC1   // empty page marker
)

func align(size int) int {
	if size > 0 {
		return ((size + 2) + PAGE - 1) &^ (PAGE - 1)
	}
	return PAGE
}

type Mapper struct {
	file   *os.File // underlying file
	data   MMap     // memory mapping
	count  int      // record count
	cursor int      // cursor
}

func OpenMapper(path string) (*Mapper, error) {
	path += `.db`
	// check to see if we need to create a new file
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// sanitize filepath
		dirs, _ := filepath.Split(path)
		// create any directories
		if err := os.MkdirAll(dirs, os.ModeDir); err != nil {
			return nil, err
		}
		// create the new file
		fd, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		// initially size it to 4MB
		if err = fd.Truncate(int64(4 * MB)); err != nil {
			return nil, err
		}
		// mark beginning of each page with empty marker
		for off := 0; off < (4 * MB); off += PAGE {
			if _, err := fd.WriteAt([]byte{EMPTY}, int64(off)); err != nil {
				log.Fatal("OpenMapper: mark beginning of each page with empty marker: ERROR BELOW\n\t%s\n", err)
				return nil, err
			}
		}
		// close file
		if err = fd.Close(); err != nil {
			return nil, err
		}
	}
	// already existing file
	fd, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, os.ModeSticky)
	if err != nil {
		return nil, err
	}
	fi, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	// map file into virtual address space
	mmap, err := MapRegion(fd, fi.Size(), RDWR, 0, 0)
	if err != nil {
		return nil, err
	}
	// create new mapper instance
	m := &Mapper{fd, mmap, 0, 0}
	// populate record count
	for int64(m.cursor) < fi.Size() {
		meta := m.GetMeta(m.cursor)
		if meta.IsEmpty {
			m.cursor++
			continue
		}
		m.count++
		m.cursor += meta.PgCount
	}
	m.cursor = 0
	return m, nil
}

func (m *Mapper) GetMeta(pos int) *struct {
	IsEmpty bool
	PgCount int
} {
	off := pos * PAGE
	if off%PAGE != 0 {
		return nil
	}
	return &struct {
		IsEmpty bool
		PgCount int
	}{
		m.data[off] == EMPTY,
		int(m.data[off+1]),
	}
}

// return offset of next available n*pages
func (m *Mapper) FindEmpty(n int) (int, bool) {
	var npages int
	for m.cursor < len(m.data) {
		if npages == n {
			return m.cursor / PAGE, true
		}
		meta := m.GetMeta(m.cursor)
		if meta.IsEmpty {
			m.cursor++
			npages++
			continue
		}
		m.cursor += meta.PgCount
		npages = 0
	}
	// NOTE: check back for empty pages
	return m.cursor / PAGE, false
}

func (m *Mapper) Read(b []byte) (int, error) {
	return -1, nil
}

// add a new record to the mapper at the first available slot
// return a non-nil error if there is an issue growing the file
func (m *Mapper) Write(b []byte) (int, error) {
	// pgs is bigger number
	pgs := align(len(b))
	pos, ok := m.FindEmpty(pgs)
	if !ok {
		if err := m.grow(); err != nil {
			return -1, err
		}
	}
	m.write(pos*PAGE, pgs, b)
	return pos, nil
}

func (m *Mapper) grow() error {
	return nil
}

func (m *Mapper) write(off, pgs int, b []byte) {
	m.data[off+1] = byte(pgs)
	copy(m.data[off+2:off+2+pgs], b)
}
