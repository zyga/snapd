// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019 Canonical Ltd
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
	"log"
	"log/syslog"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/sandbox/cgroup"
)

// NOTE: This command is not executed via "snap ...". It is only executed when
// the snap executable is invoked via the symbolic link "snapd-release-agent".
type cmdSnapdReleaseAgent struct {
	// NOTE: This is not really parsed with go-flags. See main.go for details.
	Positionals struct {
		Path string
	}
}

type logAdapter struct {
	l *log.Logger
}

func (la *logAdapter) Notice(msg string) {
	la.l.Print(msg)
}

func (la *logAdapter) Debug(msg string) {
	la.l.Print(msg)
}

func (x *cmdSnapdReleaseAgent) Execute(args []string) error {
	l, err := syslog.NewLogger(syslog.LOG_DAEMON, 0)
	if err != nil {
		return err
	}
	logger.SetLogger(&logAdapter{l: l})
	return cgroup.ReleaseAgent(l, x.Positionals.Path)
}
