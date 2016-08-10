// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
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

	"github.com/snapcore/snapd/interfaces"
)

// RebootInterface is the reboot interface for a tutorial.
type RebootInterface struct{}

// String returns the same value as Name().
func (iface *RebootInterface) Name() string {
	return "reboot"
}

// SanitizeSlot checks and possibly modifies a slot.
func (iface *RebootInterface) SanitizeSlot(slot *interfaces.Slot) error {
	if iface.Name() != slot.Interface {
		panic(fmt.Sprintf("slot is not of interface %q", iface))
	}
	// NOTE: currently we don't check anything on the slot side.
	return nil
}

// SanitizePlug checks and possibly modifies a plug.
func (iface *RebootInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface))
	}
	// NOTE: currently we don't check anything on the plug side.
	return nil
}

// ConnectedSlotSnippet returns security snippet specific to a given connection between the reboot slot and some plug.
func (iface *RebootInterface) ConnectedSlotSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecuritySecComp:
		return nil, nil
	case interfaces.SecurityDBus:
		return nil, nil
	case interfaces.SecurityUDev:
		return nil, nil
	case interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

// PermanentSlotSnippet returns security snippet permanently granted to reboot slots.
func (iface *RebootInterface) PermanentSlotSnippet(slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecuritySecComp:
		return nil, nil
	case interfaces.SecurityDBus:
		return nil, nil
	case interfaces.SecurityUDev:
		return nil, nil
	case interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

// ConnectedPlugSnippet returns security snippet specific to a given connection between the reboot plug and some slot.
func (iface *RebootInterface) ConnectedPlugSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecuritySecComp:
		return []byte(`reboot`), nil
	case interfaces.SecurityDBus:
		return nil, nil
	case interfaces.SecurityUDev:
		return nil, nil
	case interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

// PermanentPlugSnippet returns the configuration snippet required to use a reboot interface.
func (iface *RebootInterface) PermanentPlugSnippet(plug *interfaces.Plug, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecuritySecComp:
		return nil, nil
	case interfaces.SecurityDBus:
		return nil, nil
	case interfaces.SecurityUDev:
		return nil, nil
	case interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

// AutoConnect returns true if plugs and slots should be implicitly
// auto-connected when an unambiguous connection candidate is available.
//
// This interface does not auto-connect.
func (iface *RebootInterface) AutoConnect() bool {
	return false
}

func init() {
	allInterfaces = append(allInterfaces, &RebootInterface{})
}
