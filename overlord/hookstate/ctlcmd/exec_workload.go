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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/snapcore/snapd/client"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/i18n"
)

type cmdExec struct {
	baseCommand
	Workload string   `long:"workload" description:"workload name to execute" required:"yes"`
	Command  []string `positional-arg-name:"<COMMAND> [<ARGS>...]"`
}

var execCmd = addCommand("exec",
	i18n.G("Execute a workload with a specific security profile"),
	i18n.G(`
The exec command runs the given command under the specified workload security
profile. This allows confined snap processes to run workloads with different
security profiles than the parent application.
`),
	func() command { return &cmdExec{} })

func (x *cmdExec) Execute(args []string) error {
	if len(x.Command) == 0 {
		return fmt.Errorf("internal error: no command specified for exec")
	}

	// Get the snap context
	ctx, err := x.ensureContext()
	if err != nil {
		return err
	}

	// Build the snapctl exec request
	instanceName := ctx.InstanceName()
	snapName := instanceName
	if underscoreIdx := strings.Index(instanceName, "_"); underscoreIdx > 0 {
		snapName = instanceName[:underscoreIdx]
	}

	execReq := execRequest{
		Action:   "start",
		Snap:     snapName,
		Workload: x.Workload,
		Command:  x.Command,
	}

	return x.callExecAPI(execReq)
}

type execRequest struct {
	Action   string   `json:"action"`
	Cookie   string   `json:"cookie,omitempty"`
	Snap     string   `json:"snap,omitempty"`
	Workload string   `json:"workload,omitempty"`
	Command  []string `json:"command,omitempty"`
}

type execStartResponse struct {
	Cookie string `json:"cookie"`
	Scope  string `json:"scope"`
	PID    int    `json:"pid"`
	Status string `json:"status"`
}

type execStatusResponse struct {
	Cookie     string `json:"cookie"`
	Snap       string `json:"snap"`
	Workload   string `json:"workload"`
	Scope      string `json:"scope"`
	PID        int    `json:"pid"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit-code,omitempty"`
}

func (x *cmdExec) callExecAPI(req execRequest) error {
	cookie := os.Getenv("SNAPD_EXEC_COOKIE")
	if cookie != "" {
		req.Cookie = cookie
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("cannot marshal exec request: %s", err)
	}

	// Use the snapd socket to call the API
	cfg := client.Config{
		DisableAuth: true,
		Socket:      dirs.SnapSocket,
	}
	_ = client.New(&cfg)

	resp, err := http.Post("http://localhost/v2/snapctl/exec", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cannot call snapctl exec API: %s", err)
	}
	defer resp.Body.Close()

	var startResp execStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return fmt.Errorf("cannot decode exec response: %s", err)
	}

	x.printf("workload started (cookie: %s, PID: %d)\n", startResp.Cookie, startResp.PID)

	// Poll for status until completion (no timeout for long-running containers)
	return x.pollStatus(startResp.Cookie)
}

func (x *cmdExec) pollStatus(cookie string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		statusReq := execRequest{
			Action: "status",
			Cookie: cookie,
		}
		body, _ := json.Marshal(statusReq)

		resp, err := http.Post("http://localhost/v2/snapctl/exec", "application/json", bytes.NewReader(body))
		if err == nil {
			var statusResp execStatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&statusResp); err == nil {
				if statusResp.Status == "done" {
					x.printf("workload exited with code: %d\n", statusResp.ExitCode)
					resp.Body.Close()
					return nil
				}
			}
			resp.Body.Close()
		}
	}

	return nil
}
