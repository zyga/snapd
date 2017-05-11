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
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/snapcore/snapd/logger"
)

// ParseSnippet parses a seccomp filtering snippet into a sequence of rules.
func ParseSnippet(snippet string) ([]Rule, error) {
	var rules []Rule
	var comments []string

	scanner := bufio.NewScanner(strings.NewReader(snippet))
	for scanner.Scan() {
		s := scanner.Text()
		if i := strings.IndexRune(s, '#'); i != -1 {
			comments = append(comments, s[i:])
			s = s[:i]
			if s == "" {
				continue
			}
		}
		fields := strings.Fields(strings.TrimSpace(s))
		if len(fields) == 0 {
			// Ignore whitespace but collect it as a comment.
			comments = append(comments, s)
			continue
		}
		var args []ArgConstraint
		for _, field := range fields[1:] {
			var op ConstraintOp
			var value string
			switch {
			case field == "-":
				op = Any
			case strings.HasPrefix(field, "!"):
				op = NotEqual
				value = field[1:]
			case strings.HasPrefix(field, ">="):
				op = GreaterEqual
				value = field[2:]
			case strings.HasPrefix(field, "<="):
				op = LessEqual
				value = field[2:]
			case strings.HasPrefix(field, ">"):
				op = Greater
				value = field[1:]
			case strings.HasPrefix(field, "<"):
				op = Less
				value = field[1:]
			case strings.HasPrefix(field, "|"):
				op = Mask
				value = field[1:]
			default:
				op = Equal
				value = field
			}
			arg := ArgConstraint{Op: op}
			if op != Any {
				if value == "" {
					return nil, fmt.Errorf("cannot parse seccomp rule %q: expected value after operator %s", s, op)
				}
				arg.Value = value
				// Parse the value. This handles numeric literals and known symbolic constants.
				// Unparsed things are kept as-is for snap-confine to resolve.
				if resolvedValue, err := parseValue(value); err == nil {
					arg.ResolvedValue = resolvedValue
					arg.IsResolved = true
				} else {
					// Be noisy about errors while parsing.
					logger.Noticef("%s", err)
				}
			}
			args = append(args, arg)
		}
		rule := Rule{
			Comment: strings.Join(comments, "\n"),
			SysCall: SysCall(fields[0]),
			Args:    args,
		}
		comments = nil
		rules = append(rules, rule)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(comments) != 0 {
		justCommentRule := Rule{Comment: strings.Join(comments, "\n")}
		rules = append(rules, justCommentRule)
	}
	return rules, nil
}

func parseValue(s string) (int, error) {
	// value may be an interger literal.
	if value, err := strconv.Atoi(s); err == nil {
		return value, nil
	}
	// value may be a known symbolic constant.
	if value, ok := knownConstants[s]; ok {
		return value, nil
	}
	return 0, fmt.Errorf("unknown symbolic seccomp argment value %q", s)
}
