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
package snap

var systemSnap *Info = &Info{
	SuggestedName: "system",
	Version:       "1.0",
	Type:          TypeSystem,
	Architectures: []string{"all"},

	OriginalTitle:   "system",
	OriginalSummary: "the system snap represents computer resources",
	OriginalDescription: `The virtual system snap is used to represent computer resources in the form of
snap interfaces. This snap doesn't exist on disk, it's just an abstraction.
`,

	License:     "GPL-3.0",
	Confinement: StrictConfinement,
	// NOTE: slots are filled in by the interface manager on startup.
	Slots: map[string]*SlotInfo{},
	SideInfo: SideInfo{
		RealName: "system",
		Revision: R("1"),
		Channel:  "stable",
	},
}

// SystemSnap returns information about the virtual system snap.
//
// The returned object is global to all of snapd, it is never instantiated in
// another place. The object may be modified (e.g. by the interface manager)
// and the modifications are persisted during the lifetime of the process.
func SystemSnap() *Info {
	return systemSnap
}
