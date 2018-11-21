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

// Package perfstate contains internal performance monitoring parts of snapd.
package perfstate

import (
	"github.com/snapcore/snapd/overlord/state"
)

// PerformanceManager monitors performance of snapd operations inside in-memory
// ring buffer and allows exporting them for analysis.
type PerformanceManager struct {
	state *state.State
}

// Manager returns a new PerformanceManager.
func Manager(st *state.State) *PerformanceManager {
	return &PerformanceManager{state: st}
}

// Ensure implements StateManager.Ensure.
func (m *PerformanceManager) Ensure() error {
	// TODO: collect samples from snap-{run,confine,update-ns,exec}
	return nil
}

// Stop implements StateWaiterStopper.Stop.
func (m *PerformanceManager) Stop() {
	// TODO: if persistence is enabled store the ring buffer and ID counter to disk.
}
