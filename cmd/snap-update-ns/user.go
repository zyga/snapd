// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017-2019 Canonical Ltd
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

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/snap"
)

// UserProfileUpdateContext contains information about update to per-user mount namespace.
type UserProfileUpdateContext struct {
	CommonProfileUpdateContext
	// uid is the numeric user identifier associated with the user for which
	// the update operation is occurring. It may be the current UID but doesn't
	// need to be.
	uid int
}

// NewUserProfileUpdateContext returns encapsulated information for performing a per-user mount namespace update.
func NewUserProfileUpdateContext(instanceName string, fromSnapConfine bool, uid int) *UserProfileUpdateContext {
	return &UserProfileUpdateContext{
		CommonProfileUpdateContext: CommonProfileUpdateContext{
			instanceName:       instanceName,
			fromSnapConfine:    fromSnapConfine,
			currentProfilePath: currentUserProfilePath(instanceName, uid),
			desiredProfilePath: desiredUserProfilePath(instanceName),
		},
		uid: uid,
	}
}

func (upCtx *UserProfileUpdateContext) lookupUser() (*user.User, error) {
	return user.LookupId(strconv.Itoa(upCtx.uid))
}

// UID returns the user ID of the mount namespace being updated.
func (upCtx *UserProfileUpdateContext) UID() int {
	return upCtx.uid
}

// Lock acquires locks / freezes needed to synchronize mount namespace changes.
func (upCtx *UserProfileUpdateContext) Lock() (unlock func(), err error) {
	// TODO: when persistent user mount namespaces are enabled, grab a lock
	// protecting the snap and freeze snap processes here.
	return func() {}, nil
}

// Assumptions returns information about file system mutability rules.
func (upCtx *UserProfileUpdateContext) Assumptions() *Assumptions {
	// TODO: configure the secure helper and inform it about directories that
	// can be created without trespassing.
	as := &Assumptions{}
	instanceName := upCtx.InstanceName()
	as.AddUnrestrictedPaths("/tmp", "/snap/"+instanceName)
	if snapName := snap.InstanceSnap(instanceName); snapName != instanceName {
		as.AddUnrestrictedPaths("/snap/" + snapName)
	}
	if user, err := upCtx.lookupUser(); err == nil {
		as.AddUnrestrictedPaths(filepath.Join(user.HomeDir, "snap", upCtx.InstanceName()))
	}
	// TODO: Handle /home/*/snap/* when we do per-user mount namespaces and
	// allow defining layout items that refer to SNAP_USER_DATA and
	// SNAP_USER_COMMON.
	return as
}

// LoadDesiredProfile loads the desired, per-user mount profile, expanding user-specific variables.
func (upCtx *UserProfileUpdateContext) LoadDesiredProfile() (*osutil.MountProfile, error) {
	profile, err := upCtx.CommonProfileUpdateContext.LoadDesiredProfile()
	if err != nil {
		return nil, err
	}
	// TODO: when SNAP_USER_DATA, SNAP_USER_COMMON or other variables relating
	// to the user name and their home directory need to be expanded then
	// handle them here.
	expandXdgRuntimeDir(profile, upCtx.uid)
	if user, err := upCtx.lookupUser(); err == nil {
		fmt.Printf("expanding user directories\n")
		expandSnapUserDirs(profile, upCtx.InstanceName(), user.HomeDir)
	} else {
		fmt.Printf("cannot find user: %s\n", err)
	}
	return profile, nil
}

func expandSnapUserDirs(profile *osutil.MountProfile, instanceName, homeDir string) {
	rev, err := os.Readlink(filepath.Join("/snap", instanceName, "current"))
	if err != nil {
		return
	}
	snap := filepath.Join("/snap", instanceName, rev)
	snapUserData := filepath.Join(homeDir, "snap", instanceName, rev)
	snapUserCommon := filepath.Join(homeDir, "snap", instanceName, "common")
	for i := range profile.Entries {
		profile.Entries[i].Name = expandPrefixVariable(profile.Entries[i].Name, "$SNAP", snap)
		profile.Entries[i].Dir = expandPrefixVariable(profile.Entries[i].Dir, "$SNAP", snap)
		profile.Entries[i].Name = expandPrefixVariable(profile.Entries[i].Name, "$SNAP_USER_DATA", snapUserData)
		profile.Entries[i].Dir = expandPrefixVariable(profile.Entries[i].Dir, "$SNAP_USER_DATA", snapUserData)
		profile.Entries[i].Name = expandPrefixVariable(profile.Entries[i].Name, "$SNAP_USER_COMMON", snapUserCommon)
		profile.Entries[i].Dir = expandPrefixVariable(profile.Entries[i].Dir, "$SNAP_USER_COMMON", snapUserCommon)
		fmt.Printf("expanded to %s\n", profile.Entries[i])
	}
}

// SaveCurrentProfile does nothing at all.
//
// Per-user mount profiles are not persisted yet.
func (upCtx *UserProfileUpdateContext) SaveCurrentProfile(profile *osutil.MountProfile) error {
	// TODO: when persistent user mount namespaces are enabled save the
	// current, per-user mount profile here.
	return nil
}

// LoadCurrentProfile returns the empty profile.
//
// Per-user mount profiles are not persisted yet.
func (upCtx *UserProfileUpdateContext) LoadCurrentProfile() (*osutil.MountProfile, error) {
	// TODO: when persistent user mount namespaces are enabled load the
	// current, per-user mount profile here.
	return &osutil.MountProfile{}, nil
}

// desiredUserProfilePath returns the path of the fstab-like file with the desired, user-specific mount profile for a snap.
func desiredUserProfilePath(snapName string) string {
	return fmt.Sprintf("%s/snap.%s.user-fstab", dirs.SnapMountPolicyDir, snapName)
}

// currentUserProfilePath returns the path of the fstab-like file with the applied, user-specific mount profile for a snap.
func currentUserProfilePath(snapName string, uid int) string {
	return fmt.Sprintf("%s/snap.%s.%d.user-fstab", dirs.SnapRunNsDir, snapName, uid)
}
