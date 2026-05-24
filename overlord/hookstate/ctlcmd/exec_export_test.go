// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2024 Canonical Ltd
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

package ctlcmd

import (
	"encoding/json"
	"io"
)

// Export cmdExec type and methods
type ExecCmd = cmdExec

// NewExecCmd creates a new exec command with given stdout/stderr writers
func NewExecCmd(stdout, stderr io.Writer) *cmdExec {
	cmd := &cmdExec{}
	cmd.setStdout(stdout)
	cmd.setStderr(stderr)
	return cmd
}

// SetWorkload sets the workload name
func (c *cmdExec) SetWorkload(name string) {
	c.Workload = name
}

// SetCommand sets the command
func (c *cmdExec) SetCommand(cmd []string) {
	c.Command = cmd
}

// Name returns the command name
func (c *cmdExec) Name() string {
	return "exec"
}

// ShortHelp returns the short help
func (c *cmdExec) ShortHelp() string {
	return execCmd.shortHelp
}

// LongHelp returns the long help
func (c *cmdExec) LongHelp() string {
	return execCmd.longHelp
}

// Export request/response types
type (
	ExecRequest    = execRequest
	ExecResponse   = execStartResponse
	ExecStatusResp = execStatusResponse
)

// NewExecRequest creates a new exec request
func NewExecRequest(action, snap, workload string, command []string) *execRequest {
	return &execRequest{
		Action:   action,
		Snap:     snap,
		Workload: workload,
		Command:  command,
	}
}

// MarshalJSON marshals the request to JSON
func (r *execRequest) MarshalJSON() ([]byte, error) {
	type Alias execRequest
	alias := Alias(*r)
	return json.Marshal(&alias)
}

// NewExecResponse creates a new exec response
func NewExecResponse(cookie, scope string, pid int, status string) *execStartResponse {
	return &execStartResponse{
		Cookie: cookie,
		Scope:  scope,
		PID:    pid,
		Status: status,
	}
}

// NewExecStatusResponse creates a new exec status response
func NewExecStatusResponse(cookie, snap, workload, scope string, pid int, status string, exitCode int) *execStatusResponse {
	return &execStatusResponse{
		Cookie:     cookie,
		Snap:       snap,
		Workload:   workload,
		Scope:      scope,
		PID:        pid,
		Status:     status,
		ExitCode:   exitCode,
	}
}
