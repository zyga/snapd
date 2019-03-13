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

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
)

// UserProfileUpdate contains information about update to per-user mount namespace.
type UserProfileUpdate struct {
	CommonProfileUpdate
	// uid is the numeric user identifier associated with the user for which
	// the update operation is occurring. It may be the current UID but doesn't
	// need to be.
	uid int
}

// NewUserProfileUpdate returns encapsulated information for performing a per-user mount namespace update.
func NewUserProfileUpdate(instanceName string, fromSnapConfine bool, uid int) *UserProfileUpdate {
	return &UserProfileUpdate{
		CommonProfileUpdate: CommonProfileUpdate{
			instanceName:       instanceName,
			fromSnapConfine:    fromSnapConfine,
			desiredProfilePath: desiredUserProfilePath(instanceName),
		},
		uid: uid,
	}
}

// Lock acquires locks / freezes needed to synchronize mount namespace changes.
func (up *UserProfileUpdate) Lock() (unlock func(), err error) {
	// TODO: when persistent user mount namespaces are enabled, grab a lock
	// protecting the snap and freeze snap processes here.
	return func() {}, nil
}

// Assumptions returns information about file system mutability rules.
func (up *UserProfileUpdate) Assumptions() *Assumptions {
	// TODO: configure the secure helper and inform it about directories that
	// can be created without trespassing.
	as := &Assumptions{}
	// TODO: Handle /home/*/snap/* when we do per-user mount namespaces and
	// allow defining layout items that refer to SNAP_USER_DATA and
	// SNAP_USER_COMMON.
	return as
}

// LoadDesiredProfile loads the desired, per-user mount profile, expanding user-specific variables.
func (up *UserProfileUpdate) LoadDesiredProfile() (*osutil.MountProfile, error) {
	profile, err := up.CommonProfileUpdate.LoadDesiredProfile()
	if err != nil {
		return nil, err
	}
	// TODO: when SNAP_USER_DATA, SNAP_USER_COMMON or other variables relating
	// to the user name and their home directory need to be expanded then
	// handle them here.
	expandXdgRuntimeDir(profile, up.uid)
	return profile, nil
}

func applyUserFstab(up MountProfileUpdate, snapName string) error {
	desired, err := up.LoadDesiredProfile()
	if err != nil {
		return fmt.Errorf("cannot load desired user mount profile of snap %q: %s", snapName, err)
	}
	debugShowProfile(desired, "desired mount profile")
	as := up.Assumptions()
	_, err = applyProfile(up, snapName, &osutil.MountProfile{}, desired, as)
	return err
}

// desiredUserProfilePath returns the path of the fstab-like file with the desired, user-specific mount profile for a snap.
func desiredUserProfilePath(snapName string) string {
	return fmt.Sprintf("%s/snap.%s.user-fstab", dirs.SnapMountPolicyDir, snapName)
}
