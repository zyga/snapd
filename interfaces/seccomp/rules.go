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
	"bytes"
	"fmt"
)

type formatFlags int

const (
	// resolveSymbols makes us resolve stuff like AF_UNIX into numberic constants.
	resolveSymbols formatFlags = 1 << iota
)

// Rule represents a seccomp filtering rule.
//
// Rules specify the name of the system call and optionally constraints on
// integer-typed arguments. If argument constraints are given they must have
// the same cardinality as the system call itself. If omitted seccomp behaves
// as if the "-" constraint was given.
type Rule struct {
	SysCall SysCall
	Args    []ArgConstraint
	Comment string
}

// String converts a seccomp filtering rule to a string.
//
// The string is in a format that is compatible with snap-confine's seccomp
// parser. This format is internal to snapd and may change at any time.
func (r Rule) String() string {
	buf := bytes.NewBuffer(nil)
	if r.Comment != "" {
		buf.WriteString(r.Comment)
		buf.WriteRune('\n')
	}
	symbolic := r.formatRule(0)
	numeric := r.formatRule(resolveSymbols)
	if symbolic != numeric {
		// Add the symbolic representation as a comment
		// if we have resolved anything (for debugging)
		buf.WriteString("# resolved by snapd, was: ")
		buf.WriteString(symbolic)
	}
	buf.WriteString(numeric)
	return buf.String()
}

func (r Rule) formatRule(flags formatFlags) string {
	buf := bytes.NewBuffer(nil)
	if r.SysCall != "" {
		buf.WriteString(string(r.SysCall))
		for _, arg := range r.Args {
			buf.WriteRune(' ')
			buf.WriteString(arg.formatConstraint(flags))
		}
		buf.WriteRune('\n')
	}
	return buf.String()
}

// ArgConstraint represents a constraint on a positional argument to a system call.
//
// constraint can take one of the following forms:
// "-"       - argument may have any value.
// "VALUE"   - argument must be equal to VALUE.
// "!VALUE"  - argument must be not equal to VALUE.
// ">=VALUE" - argument must be greater than or equal to VALUE.
// "<=VALUE" - argument must be less than or equal to VALUE.
// ">VALUE"  - argument must be greater than VALUE.
// "<VALUE"  - argument must be less than VALUE.
// "|VALUE"  - argument must be non-zero when masked with bitwise-AND VALUE.
//
// The value may be resolved or not. Unresolved value are passed symbolically
// and snap-confine must perform the lookup. Resolved value are passed as a
// numeric constant.
type ArgConstraint struct {
	Op            ConstraintOp
	Value         string
	ResolvedValue int
	IsResolved    bool
}

// String converts an argument constraint into a format understood by snap-confine.
func (ac ArgConstraint) String() string {
	return ac.formatConstraint(0)
}

// String converts an argument constraint into a format understood by snap-confine.
func (ac ArgConstraint) formatConstraint(flags formatFlags) string {
	switch ac.Op {
	case Any:
		return ac.Op.String()
	default:
		if flags&resolveSymbols != 0 && ac.IsResolved {
			return fmt.Sprintf("%s%d", ac.Op, ac.ResolvedValue)
		}
		return fmt.Sprintf("%s%s", ac.Op, ac.Value)
	}
}

// ConstraintOp represents an operator in an argument constraint.
type ConstraintOp int

const (
	// Any indicates that syscall argument may have any value.
	Any ConstraintOp = iota
	// Equal indicates that syscall argument must have a specific value.
	Equal
	// NotEqual indicates that the syscall argument must not have a specific value.
	NotEqual
	// GreaterEqual indicates that syscall argument must be greater than or equal to a specific value.
	GreaterEqual
	// LessEqual indicates that syscall argument must be less than or equal to a specific value.
	LessEqual
	// Greater indicates that syscall argument must be greater than a specific value.
	Greater
	// Less indicates that syscall argument must be less than a specific value.
	Less
	// Mask indicates that syscall argument must have a non-zero bitwise intersection with a specific mask.
	Mask
)

// String converts an constraint operator into a format understood by snap-confine.
func (op ConstraintOp) String() string {
	switch op {
	case Any:
		return "-"
	case Equal:
		return ""
	case NotEqual:
		return "!"
	case GreaterEqual:
		return ">="
	case LessEqual:
		return "<="
	case Greater:
		return ">"
	case Less:
		return "<"
	case Mask:
		return "|"
	}
	panic(fmt.Errorf("unexpected seccomp argument constraint operator %d", op))
}
