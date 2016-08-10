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
	"github.com/snapcore/snapd/interfaces"
)

// NewRebootInterface returns a new "reboot" interface.
func NewRebootInterface() interfaces.Interface {
	return &commonInterface{
		name: "reboot",
		connectedPlugAppArmor: `capability sys_boot,`,
		connectedPlugSecComp:  `reboot`,
		reservedForOS:         true,
	}
}

func init() {
	allInterfaces = append(allInterfaces, NewRebootInterface())
}
