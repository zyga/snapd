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

package main

import (
	"github.com/snapcore/snapd/i18n"

	"fmt"
	"time"

	"github.com/jessevdk/go-flags"
)

var shortSpeedHelp = i18n.G("Print internal performance measurements")
var longSpeedHelp = i18n.G(`
The speed command prints internal performance measurements. Measurements
can be sorted, filtered or aggregated using optional arguments.
`)

type cmdSpeed struct {
	clientMixin
}

func init() {
	addDebugCommand("speed", shortSpeedHelp, longSpeedHelp, func() flags.Commander {
		return &cmdSpeed{}
	}, nil, nil)
}

type Sample struct {
	ID        uint64    `json:"id"`
	StartTime time.Time `json:"start-time"`
	EndTime   time.Time `json:"end-time"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	TaskID    string    `json:"task-id,omitempty"`
	ChangeID  string    `json:"change-id,omitempty"`
	SnapName  string    `json:"snap-name,omitempty"`
	ManagerID string    `json:"manager,omitempty"`
	MiscID    string    `json:"misc-id,omitempty"`
}

func (s Sample) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

func (cmd cmdSpeed) Execute(args []string) error {
	if len(args) > 0 {
		return ErrExtraArgs
	}

	var samples []Sample
	err := cmd.client.Debug("speed", nil, &samples)
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		fmt.Fprintf(Stdout, "There are no performance measurements yet\n")
		return nil
	}
	w := tabWriter()
	fmt.Fprintf(w, "ID\tKind\tDuration*\tName\tTaskID\tChangeID\tSnapName\tManagerID\tMiscID\n")
	for _, s := range samples {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.ID, s.Kind, s.Duration().Round(1*time.Millisecond), s.Name,
			s.TaskID, s.ChangeID, s.SnapName, s.ManagerID, s.MiscID)
	}
	w.Flush()
	return nil
}
