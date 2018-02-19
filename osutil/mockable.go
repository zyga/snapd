// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package osutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/snapcore/snapd/osutil/sys"
)

var (
	procSelfMountInfo = func() string { return ProcSelfMountInfo }
	etcFstab          = "/etc/fstab"
)

// For mocking everything during testing.
var (
	osLstat    = os.Lstat
	osReadlink = os.Readlink
	osSymlink  = os.Symlink
	osRemove   = os.Remove

	sysClose   = syscall.Close
	sysMkdirat = syscall.Mkdirat
	sysMount   = syscall.Mount
	sysOpen    = syscall.Open
	sysOpenat  = syscall.Openat
	sysUnmount = syscall.Unmount
	sysFstat   = syscall.Fstat
	sysFchown  = sys.Fchown

	ioutilReadDir = ioutil.ReadDir
)

//MockMountInfo mocks content of /proc/self/mountinfo read by IsHomeUsingNFS
func MockMountInfo(text string) (restore func()) {
	old := procSelfMountInfo
	f, err := ioutil.TempFile("", "mountinfo")
	if err != nil {
		panic(fmt.Errorf("cannot open temporary file: %s", err))
	}
	new := f.Name()
	if err := ioutil.WriteFile(new, []byte(text), 0644); err != nil {
		panic(fmt.Errorf("cannot write mock mountinfo file: %s", err))
	}
	procSelfMountInfo = func() string { return new }
	return func() {
		os.Remove(new)
		procSelfMountInfo = old
	}
}

//MockMountInfoVary mocks content of /proc/self/mountinfo for subsequent inspections.
func MockMountInfoVary(texts ...string) (restore func()) {
	old := procSelfMountInfo
	fakes := make([]string, 0, len(texts))
	for _, text := range texts {
		f, err := ioutil.TempFile("", "mountinfo")
		if err != nil {
			panic(fmt.Errorf("cannot open temporary file: %s", err))
		}
		fname := f.Name()
		if err := ioutil.WriteFile(fname, []byte(text), 0644); err != nil {
			panic(fmt.Errorf("cannot write mock mountinfo file: %s", err))
		}
		fakes = append(fakes, fname)
	}
	var i int
	procSelfMountInfo = func() string {
		if i < len(fakes) {
			fake := fakes[i]
			i++
			return fake
		}
		panic("ran out of fake files for /proc/self/mountinfo")
	}
	return func() {
		for _, fname := range fakes {
			os.Remove(fname)
		}
		procSelfMountInfo = old
	}
}

// MockEtcFstab mocks content of /etc/fstab read by IsHomeUsingNFS
func MockEtcFstab(text string) (restore func()) {
	old := etcFstab
	f, err := ioutil.TempFile("", "fstab")
	if err != nil {
		panic(fmt.Errorf("cannot open temporary file: %s", err))
	}
	if err := ioutil.WriteFile(f.Name(), []byte(text), 0644); err != nil {
		panic(fmt.Errorf("cannot write mock fstab file: %s", err))
	}
	etcFstab = f.Name()
	return func() {
		if etcFstab == "/etc/fstab" {
			panic("respectfully refusing to remove /etc/fstab")
		}
		os.Remove(etcFstab)
		etcFstab = old
	}
}
