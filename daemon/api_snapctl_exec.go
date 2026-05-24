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

package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/i18n"
	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/overlord/auth"
	"github.com/snapcore/snapd/overlord/snapstate"
	"github.com/snapcore/snapd/randutil"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snapdtool"
	"github.com/snapcore/snapd/systemd"
)

// execJob tracks the state of a workload execution job
type execJob struct {
	Cookie       string
	ScopeName    string
	SnapInstance string
	WorkloadName string
	Command      []string
	UID          uint32
	Created      time.Time
	PID          int
	ExitCode     *int
}

// execJobs manages the lifecycle of workload execution jobs
type execJobs struct {
	mu   sync.RWMutex
	jobs map[string]*execJob
}

var execJobsManager = &execJobs{
	jobs: make(map[string]*execJob),
}

// EnsureSnapctlExecDir creates the directory for snapctl exec state files
func EnsureSnapctlExecDir() error {
	dir := dirs.SnapdSnapctlExecDir()
	return os.MkdirAll(dir, 0700)
}

// execRequest defines the command structure for starting a workload via snapctl
type execRequest struct {
	Action   string   `json:"action"`
	Cookie   string   `json:"cookie,omitempty"`
	Snap     string   `json:"snap,omitempty"`
	Workload string   `json:"workload,omitempty"`
	Command  []string `json:"command,omitempty"`
}

var (
	snapctlExecCmd = &Command{
		Path:        "/v2/snapctl/exec",
		POST:        runSnapctlExec,
		Actions:     []string{"start", "status", "cancel", "help"},
		WriteAccess: snapAccess{},
	}

	snapctlExecSpawn         = spawnWorkloadScope
	snapctlExecReadExitCode  = readExitCode
	snapctlExecWriteExitCode = writeExitCode
)

func runSnapctlExec(c *Command, r *http.Request, user *auth.UserState) Response {
	var execReq execRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&execReq); err != nil {
		return BadRequest("cannot decode exec request: %s", err)
	}

	ucred, err := ucrednetGet(r.RemoteAddr)
	if err != nil {
		return Forbidden("cannot get remote user: %s", err)
	}

	contextID := r.Header.Get("X-Snapd-Context-ID")
	if contextID == "" && execReq.Action != "help" && execReq.Action != "" {
		return BadRequest("snapctl exec requires a snap context")
	}

	switch execReq.Action {
	case "start", "":
		return startWorkload(c, execReq, ucred.Uid, contextID)
	case "status":
		return getWorkloadStatus(execReq.Cookie)
	case "cancel":
		return cancelWorkload(execReq.Cookie)
	case "help":
		return helpExec()
	default:
		return BadRequest("unknown exec action: %s", execReq.Action)
	}
}

func helpExec() Response {
	help := map[string]string{
		"summary":  i18n.G("Execute a workload with a specific security profile"),
		"strategy": i18n.G("Actions: start, status, cancel, help"),
	}
	return SyncResponse(help)
}

func startWorkload(c *Command, req execRequest, uid uint32, contextID string) Response {
	if req.Snap == "" {
		return BadRequest("snap name is required")
	}
	if req.Workload == "" {
		return BadRequest("workload name is required")
	}
	if len(req.Command) == 0 {
		return BadRequest("command is required")
	}

	cookie := req.Cookie
	if cookie == "" {
		cookie = generateCookie()
	}

	job := &execJob{
		Cookie:       cookie,
		SnapInstance: req.Snap,
		WorkloadName: req.Workload,
		Command:      req.Command,
		UID:          uid,
		Created:      time.Now(),
	}

	execJobsManager.mu.Lock()
	execJobsManager.jobs[cookie] = job
	execJobsManager.mu.Unlock()

	exitFilePath := filepath.Join(dirs.SnapdSnapctlExecDir(), cookie+".exit")
	if err := writeExitCode(exitFilePath, nil); err != nil {
		logger.Noticef("cannot create exit code file: %s", err)
	}

	scopeName, pid, err := snapctlExecSpawn(c.d, job)
	if err != nil {
		return BadRequest("cannot spawn workload: %s", err)
	}

	job.ScopeName = scopeName
	job.PID = pid

	go monitorJob(cookie, pid)

	result := map[string]any{
		"cookie":   cookie,
		"scope":    scopeName,
		"pid":      pid,
		"status":   "running",
	}

	// Return 202 Accepted for async operation
	return &respJSON{
		Type:   ResponseTypeAsync,
		Status: 202,
		Result: result,
	}
}

func getWorkloadStatus(cookie string) Response {
	execJobsManager.mu.RLock()
	job, exists := execJobsManager.jobs[cookie]
	execJobsManager.mu.RUnlock()

	if !exists {
		return BadRequest("unknown cookie: %s", cookie)
	}

	result := map[string]any{
		"cookie":     cookie,
		"snap":       job.SnapInstance,
		"workload":   job.WorkloadName,
		"scope":      job.ScopeName,
		"pid":        job.PID,
		"status":     "running",
	}

	exitCode, err := snapctlExecReadExitCode(filepath.Join(dirs.SnapdSnapctlExecDir(), cookie+".exit"))
	if err == nil && exitCode != nil {
		result["status"] = "done"
		result["exit-code"] = *exitCode
	}

	return SyncResponse(result)
}

func cancelWorkload(cookie string) Response {
	execJobsManager.mu.RLock()
	job, exists := execJobsManager.jobs[cookie]
	execJobsManager.mu.RUnlock()

	if !exists {
		return BadRequest("unknown cookie: %s", cookie)
	}

	if job.PID > 0 {
		syscall.Kill(job.PID, syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		syscall.Kill(job.PID, syscall.SIGKILL)
	}

	execJobsManager.mu.Lock()
	delete(execJobsManager.jobs, cookie)
	execJobsManager.mu.Unlock()

	return SyncResponse(map[string]any{
		"cookie": cookie,
		"status": "cancelled",
	})
}

func monitorJob(cookie string, pid int) {
	defer func() {
		execJobsManager.mu.Lock()
		delete(execJobsManager.jobs, cookie)
		execJobsManager.mu.Unlock()
	}()

	var status syscall.WaitStatus
	pid2, err := syscall.Wait4(pid, &status, 0, nil)
	if err != nil || pid2 != pid {
		exitCode := 137
		_ = snapctlExecWriteExitCode(filepath.Join(dirs.SnapdSnapctlExecDir(), cookie+".exit"), &exitCode)
		return
	}

	exitCode := status.ExitStatus()
	if status.Signaled() {
		exitCode = 128 + int(status.Signal())
	}

	_ = snapctlExecWriteExitCode(filepath.Join(dirs.SnapdSnapctlExecDir(), cookie+".exit"), &exitCode)
}

func spawnWorkloadScope(d *Daemon, job *execJob) (scopeName string, pid int, err error) {
	snapInstance := job.SnapInstance
	workloadName := job.WorkloadName

	st := d.overlord.State()
	info, err := snapstate.CurrentInfo(st, snapInstance)
	if err != nil {
		return "", 0, fmt.Errorf("cannot get snap info: %s", err)
	}

	securityTag := snap.WorkloadSecurityTag(snapInstance, workloadName)

	uuid, err := randutil.RandomKernelUUID()
	if err != nil {
		return "", 0, fmt.Errorf("cannot generate UUID: %s", err)
	}

	unitName, err := systemd.SecurityTagToUnitName(securityTag)
	if err != nil {
		return "", 0, fmt.Errorf("cannot convert security tag to unit name: %s", err)
	}
	scopeName = fmt.Sprintf("%s-%s.scope", unitName, uuid)

	snapConfineWorkload, err := snapdtool.InternalToolPath("snap-confine-workload")
	if err != nil {
		return "", 0, fmt.Errorf("cannot find snap-confine-workload: %s", err)
	}

	workloadApp := fmt.Sprintf("%s.%s", snapInstance, workloadName)

	cmd := append([]string{snapConfineWorkload, securityTag, workloadApp}, job.Command...)

	// Get the home directory for the user running the process
	usr, _ := user.Current()
	home := usr.HomeDir
	opts := &dirs.SnapDirOptions{}

	procAttr := &syscall.ProcAttr{
		Dir: "/",
		Env: []string{
			"SNAP_INSTANCE_NAME=" + snapInstance,
			"SNAP_NAME=" + info.SnapName(),
			"SNAP_REVISION=" + info.Revision.String(),
			"SNAP_COMMON=" + info.CommonDataDir(),
			"SNAP_DATA=" + info.DataDir(),
			"SNAP_USER_DATA=" + info.UserDataDir(home, opts),
			"SNAP_USER_COMMON=" + info.UserCommonDataDir(home, opts),
		},
		Files: []uintptr{0, 1, 2},
	}

	pid, err = syscall.ForkExec(cmd[0], cmd, procAttr)
	if err != nil {
		return "", 0, fmt.Errorf("cannot fork exec snap-confine-workload: %s", err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		logger.Debugf("workload scope %s created for PID %d", scopeName, pid)
	}()

	return scopeName, pid, nil
}

func generateCookie() string {
	return fmt.Sprintf("%x", time.Now().UnixNano()) + fmt.Sprintf("%x", os.Getpid())
}

func readExitCode(path string) (*int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func writeExitCode(path string, code *int) error {
	if code == nil {
		return os.WriteFile(path, []byte{}, 0644)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(*code)), 0644)
}
