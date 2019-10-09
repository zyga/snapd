// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019 Canonical Ltd
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

package cgroup

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/snapcore/snapd/osutil"
)

// splitCgroupPath analyzes and splits the cgroup path into a snap name and security tag.
//
// Given a typical group path of "/snap.pkg-name.app-name" or
// "/snap.pkg-name.hooks.hook-name", the function returns "pkg-name" along with
// "snap.pkg-name.app-name" or "snap.pkg-name.hooks.hook-name" respectively.
//
// If the cgroup path is not like one of those then an error is returned.
func splitCgroupPath(cgroupPath string) (snapName, snapSecurityTag string, err error) {
	if !strings.HasPrefix(cgroupPath, "/") {
		return "", "", fmt.Errorf("cgroup path is not absolute")
	}
	cgroupPath = cgroupPath[1:]
	if !strings.HasPrefix(cgroupPath, "snap.") {
		return "", "", fmt.Errorf("cgroup path unrelated to snaps")
	}
	if strings.IndexRune(cgroupPath, '/') != -1 {
		return "", "", fmt.Errorf("cgroup path describes sub-hierarchy")
	}
	if n := strings.Count(cgroupPath, "."); n < 2 || n > 3 {
		return "", "", fmt.Errorf("cgroup path is not a snap security tag")
	}
	parts := strings.SplitN(cgroupPath, ".", 3)
	return parts[1], cgroupPath, nil
}

// removeSnapdCgroupHierarchy removes the hierarchy from /run/snapd/cgroup.
//
// The hierarchy is created by snap-confine. It is called "SNAP_SECURITY_TAG".
func removeSnapdCgroupHierarchy(logger *log.Logger, snapSecurityTag string) error {
	fname := filepath.Join("/run/snapd/cgroup", snapSecurityTag)
	if err := os.Remove(fname); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	logger.Printf("removed cgroup hierarchy %s", fname)
	return nil
}

// removeFreezerCgroupHierarchy removes the hierarchy from /sys/fs/cgroup/freezer.
//
// The hierarchy is created by snap-confine. It is called "snap.$SNAP_NAME".
func removeFreezerCgroupHierarchy(logger *log.Logger, snapName string) error {
	fname := filepath.Join("/sys/fs/cgroup/freezer", "snap."+snapName)
	if err := os.Remove(fname); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	logger.Printf("removed cgroup hierarchy %s", fname)
	return nil
}

// removeDevicesCgroupHierarchy removes the hierarchy from /sys/fs/cgroup/devices.
//
// The hierarchy is created by snap-device-helper. It is called
// "SNAP_SECURITY_TAG".
func removeDevicesCgroupHierarchy(logger *log.Logger, snapSecurityTag string) error {
	fname := filepath.Join("/sys/fs/cgroup/devices", snapSecurityTag)
	if err := os.Remove(fname); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	logger.Printf("removed cgroup hierarchy %s", fname)
	return nil
}

// removeMonitorFile removes the monitor file from /run/snapd/montor.
//
// The monitor file is $SNAP_SECURITY_TAG and is written by snap-confine.
// While the file exists there are likely processes running that belong to that
// security tag.
func removeMonitorFile(logger *log.Logger, snapSecurityTag string) error {
	fname := filepath.Join("/run/snapd/monitor", snapSecurityTag)
	if err := os.Remove(fname); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	logger.Printf("removed monitor file %s", fname)
	return nil
}

// isSnapInstalled reeturns true if a snap is still installed in the system.
//
// For the purpose of the logic here, a snap is considered to be installed if
// there's a non-empty mount directory for the snap in /snap or
// /var/lib/snapd/snap.
//
// Note that an empty directry named after the non-instance snap always exists
// when a snap with an instance key is installed.
func isSnapInstalled(snapName string) bool {
	for _, fname := range []string{
		filepath.Join("/snap", snapName),
		filepath.Join("/var/lib/snapd/snap", snapName),
	} {
		files, err := ioutil.ReadDir(fname)
		if err == nil && len(files) > 0 {
			return true
		}
	}
	return false
}

// unloadAppArmorProfiles unloads apparmor profiles belonging to the given snap.
//
// The logic assumes that the snap is now installed. Removing the profiles used
// by existing processes unfortunately makes them unconfined. The logic removes
// the per-snap snap-update-ns profile as well as the per-security-tag
// snap-specific profiles.
func unloadAppArmorProfiles(logger *log.Logger, snapName string) error {
	kernelIf := "/sys/kernel/security/apparmor"
	files, err := ioutil.ReadDir(filepath.Join(kernelIf, "policy/profiles"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	// Enumerate profiles in memory. Each profile is represented by a
	// directory with a name related to the profile name.
	for _, fi := range files {
		if !fi.IsDir() {
			continue
		}
		// We want to consider directory entires "snap.foo.*" and
		// "snap-update-ns.foo.*". Note that the wildcard stands both
		// for the usual security tag elements (app-name or
		// hook.hook-name) but also, most importantly, for the ".N"
		// suffix where "N" is the kernel-assigned identifier of a
		// particular iteration of the profile.
		if name := fi.Name(); !strings.HasPrefix(name, "snap."+snapName+".") && !strings.HasPrefix(name, "snap-update-ns."+snapName+".") {
			continue
		}
		// Read the actual name of the profile. This name is different
		// from the directory name itself as it doesn't contain the
		// iteration number.
		name, err := ioutil.ReadFile(filepath.Join(kernelIf, "policy/profiles", fi.Name(), "name"))
		if err != nil {
			return err
		}
		// The kernel interface takes a NUL terminated string. Replace
		// the trailing newline with NUL and write a single request to
		// the kernel apparmor interface via the ".remove" file.
		nameLen := len(name)
		if nameLen == 0 || name[nameLen-1] != '\n' {
			continue
		}
		name[nameLen-1] = 0
		if err := ioutil.WriteFile(filepath.Join(kernelIf, ".remove"), name, 0666); err != nil {
			return err
		}
		logger.Printf("unloaded apparmor profile %q", name[:nameLen-1])
	}
	return nil
}

// ReleaseAgent implements snapd-release-agent flow.
func ReleaseAgent(logger *log.Logger, cgroupPath string) error {
	logger.Printf("snapd-release-agent invoked for %q", cgroupPath)
	snapName, snapSecurityTag, err := splitCgroupPath(cgroupPath)
	if err != nil {
		return err
	}

	// Grab the snap lock
	if err := os.MkdirAll("/run/snapd/lock", 0755); err != nil {
		return err
	}
	lock, err := osutil.NewFileLock(filepath.Join("/run/snapd/lock", snapName+".lock"))
	if err != nil {
		return err
	}
	defer lock.Close()
	lock.Lock()
	defer lock.Unlock()

	// Perform all the cleanup
	if err := removeSnapdCgroupHierarchy(logger, snapSecurityTag); err != nil {
		logger.Fatal(err)
	}
	if err := removeFreezerCgroupHierarchy(logger, snapName); err != nil {
		logger.Fatal(err)
	}
	if err := removeDevicesCgroupHierarchy(logger, snapSecurityTag); err != nil {
		logger.Fatal(err)
	}
	if !isSnapInstalled(snapName) {
		if err := unloadAppArmorProfiles(logger, snapName); err != nil {
			logger.Fatal(err)
		}
	}
	if err := removeMonitorFile(logger, snapSecurityTag); err != nil {
		logger.Fatal(err)
	}

	return nil
}
