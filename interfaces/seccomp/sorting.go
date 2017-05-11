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

package seccomp

import (
	"strings"
)

type bySysCall []Rule

func (c bySysCall) Len() int      { return len(c) }
func (c bySysCall) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c bySysCall) Less(i, j int) bool {
	return compareRules(c[i], c[j]) < 0
}

func compareRules(a, b Rule) int {
	// Sort by system call name.
	if r := strings.Compare(string(a.SysCall), string(b.SysCall)); r != 0 {
		return r
	}
	// Sort by individual argument constraints.
	for i := 0; i < len(a.Args) && i < len(b.Args); i++ {
		if r := compareArgConstraints(a.Args[i], b.Args[i]); r != 0 {
			return r
		}
	}
	// Sort by number of argument constraints.
	if len(a.Args) != len(b.Args) {
		if len(a.Args) < len(b.Args) {
			return -1
		}
		return +1
	}
	return 0
}

// compareArgConstraints compares two argument constraints and returns -1, 0, or +1.
func compareArgConstraints(a, b ArgConstraint) int {
	// Sort by argument constraint operators.
	if a.Op != b.Op {
		if a.Op < b.Op {
			return -1
		}
		return +1
	}
	// The "any" operator does not take values.
	if a.Op == Any && b.Op == Any {
		return 0
	}
	// All other operators take an argument, it may be resolved (preferred) or not.
	switch {
	case a.IsResolved && b.IsResolved:
		// Sort by resolved argument constraint if both are resolved.
		if a.ResolvedValue != b.ResolvedValue {
			if a.ResolvedValue < b.ResolvedValue {
				return -1
			}
			return +1
		}
	case !a.IsResolved && !b.IsResolved:
		// Sort by unresolved argument value if both are unresolved.
		if a.Value != b.Value {
			if a.Value < b.Value {
				return -1
			}
			return +1
		}
	default:
		// Sort resolved elements before unresolved elements.
		if a.IsResolved {
			return -1
		}
		return +1
	}
	return 0
}
