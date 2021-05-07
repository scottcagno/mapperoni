/*
 * // Copyright (c) 2021. Scott Cagno. All rights reserved.
 * // Use of this source code is governed by a BSD-style (clause 3)
 * // license that can be found in the root of this project in the LICENSE file.
 */

package mmap

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
)

type MappedFile struct {
	fd   *os.File
	data MMap
	cur  int64
}

func OpenFile(path string) (*os.File, error) {
	var fd *os.File
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		dir, file := filepath.Split(path)
		err = os.MkdirAll(dir, os.ModeDir)
		if err != nil {
			return nil, err
		}
		fd, err = os.Create(dir + file)
		if err != nil {
			return nil, err
		}
		err = fd.Close()
		if err != nil {
			return fd, err
		}
	}
	// removing os.O_APPEND for now
	fd, err = os.OpenFile(path, os.O_RDWR|os.O_TRUNC, os.ModeSticky)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func OpenMappedFile(path string, prot, flags int) (*MappedFile, error) {
	fd, err := OpenFile(path)
	if err != nil {
		return nil, err
	}
	mm, err := Map(fd, prot, flags)
	if err != nil {
		return nil, err
	}
	mf := &MappedFile{
		fd:   fd,
		data: mm,
	}
	return mf, nil
}

func (mf *MappedFile) Read(b []byte) (int, error) {
	return -1, nil
}

func (mf *MappedFile) ReadAt(b []byte, off int64) (int, error) {
	return -1, nil
}

func (mf *MappedFile) Write(b []byte) (int, error) {
	return -1, nil
}

func (mf *MappedFile) WriteAt(b []byte, off int64) (int, error) {
	return -1, nil
}

func (mf *MappedFile) Seek(offset int64, whence int) (int64, error) {
	if mf == nil {
		return -1, os.ErrInvalid
	}
	switch whence {
	case io.SeekStart:
		mf.cur = offset
	case io.SeekCurrent:
		mf.cur += offset
	case io.SeekEnd:
		mf.cur = int64(len(mf.data)) - offset
	default:
		return -1, os.ErrInvalid
	}
	if mf.cur < 0 {
		return -1, bufio.ErrNegativeCount
	}
	return mf.cur, nil
}

func (mf *MappedFile) Len() int {
	return len(mf.data)
}

func (mf *MappedFile) Close() error {
	err := mf.data.Flush()
	if err != nil {
		return err
	}
	err = mf.data.Unmap()
	if err != nil {
		return err
	}
	err = mf.fd.Close()
	if err != nil {
		return err
	}
	return nil
}
