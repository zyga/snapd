// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
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

package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/mount"
	"github.com/snapcore/snapd/snap"
)

const overmountDescription = `
The overmount interface allows snaps to freely shape their mount namespace.
`

// TODO: move this to the forum once the interface definition crystalizes.
const overmountDescriptionDevel = `
Snaps can declare the overmount interface by defining a plug with one
attribute, entries, that holds a list of fstab-like entries. Each entry
describes a single new mount operation.

For security only the tmpfs filesystem and bind mounts are supported. Certain
directories are forbidden and cannot be used as mount sources or targets.
Currently those are just /etc, /media and /home.

Within each entry certain variables are expanded to their typical values. Those
variables are: SNAP, SNAP_DATA, SNAP_COMMON, SNAP_NAME and SNAP_REVISION.

The target directory is made automatically if possible. Keep in mind that vast
majority of filesystem locations are read-only (with the notable exception of
$SNAP_DATA and $SNAP_COMMON). To work around that you may choose to mount a
tmpfs (which is an in-memory, non-persistent filesystem) first. This is
illustrated by the example below where /srv/data is created automatically,
after mounting a tmpfs on /srv and before bind-mounting $SNAP_DATA to
/srv/data.

plugs:
  overmount:
    entries:
      - "none /srv/website tmpfs defaults 0 0"
      - "$SNAP_DATA /srv/data none bind,rw 0 0"
`

// overmountInterface allows sharing content between snaps
type overmountInterface struct{}

func (iface *overmountInterface) Name() string {
	return "overmount"
}

func (iface *overmountInterface) MetaData() interfaces.MetaData {
	return interfaces.MetaData{
		Description:       overmountDescription,
		ImplicitOnCore:    true,
		ImplicitOnClassic: true,
	}
}

func (iface *overmountInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface.Name()))
	}
	entries, err := iface.mountEntries(plug)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("overmount plug without a list of entries makes no sense")
	}
	return nil
}

func (iface *overmountInterface) SanitizeSlot(slot *interfaces.Slot) error {
	if iface.Name() != slot.Interface {
		panic(fmt.Sprintf("slot is not of interface %q", iface.Name()))
	}
	if slot.Snap.Type != snap.TypeOS {
		return fmt.Errorf("%s slots are reserved for the operating system snap", iface.Name())
	}
	return nil
}

func (iface *overmountInterface) AutoConnect(plug *interfaces.Plug, slot *interfaces.Slot) bool {
	// allow what declarations allowed
	return true
}

func (iface *overmountInterface) AppArmorConnectedPlug(spec *apparmor.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	entries, err := iface.mountEntries(plug)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// Allow all kind of access to each directory.
		// XXX: This may seem overly open but we already reject access to
		// things visible in the external mount namespace, to /run and to
		// /home.
		spec.AddSnippet(fmt.Sprintf("%s/** mrwklix,\n", entry.Dir))
	}
	return nil
}

func (iface *overmountInterface) MountConnectedPlug(spec *mount.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	entries, err := iface.mountEntries(plug)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := spec.AddMountEntry(entry); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	registerIface(&overmountInterface{})
}

// internal helpers

func (iface *overmountInterface) mountEntries(plug *interfaces.Plug) ([]mount.Entry, error) {
	var entries []mount.Entry
	rawEntries, ok := plug.Attrs["entries"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("overmount entries must be a list of strings")
	}
	for _, rawEntry := range rawEntries {
		entryText, ok := rawEntry.(string)
		if !ok {
			return nil, fmt.Errorf("overmount entry %q is not a string", rawEntry)
		}
		entry, err := mount.ParseEntry(entryText)
		if err != nil {
			return nil, fmt.Errorf("overmount entry %q cannot be parsed: %s", entryText, err)
		}
		if err := validateOvermountEntry(entry); err != nil {
			return nil, fmt.Errorf("overmount entry %q cannot be used: %s", entryText, err)
		}
		entry.Name = filepath.Clean(resolveSpecialVariable(entry.Name, plug.Snap, resolveFlags(0)))
		entry.Dir = filepath.Clean(resolveSpecialVariable(entry.Dir, plug.Snap, resolveFlags(0)))
		entries = append(entries, entry)
	}
	return entries, nil
}

func validateOvermountEntry(entry mount.Entry) error {
	// TODO: look at $... and ${...} variables and validate those.
	for _, forbidden := range []string{"/run", "/media", "/home"} {
		if strings.HasPrefix(entry.Name, forbidden) || strings.HasPrefix(entry.Dir, forbidden) {
			return fmt.Errorf("mounting in %q is not allowed", forbidden)
		}
	}
	if entry.Type != "tmpfs" && entry.Type != "none" {
		return fmt.Errorf("filesystem type %q is not allowed", entry.Type)
	}
	opts, err := mount.OptsToFlags(entry.Options)
	if err != nil {
		return err
	}
	if opts & ^(syscall.MS_BIND|syscall.MS_RDONLY) != 0 {
		return fmt.Errorf("mount options %q are not allowed", strings.Join(entry.Options, ","))
	}
	if entry.DumpFrequency != 0 {
		return fmt.Errorf("dump frequency must be zero")
	}
	if entry.CheckPassNumber != 0 {
		return fmt.Errorf("filesystem check pass number must be zero")
	}
	return nil
}
