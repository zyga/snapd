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

// Exported for testing

var (
	GenerateCookie  = generateCookie
	ReadExitCode    = readExitCode
	WriteExitCode   = writeExitCode
	AddExecJob      = func(cookie string, job *execJob) {
		execJobsManager.mu.Lock()
		execJobsManager.jobs[cookie] = job
		execJobsManager.mu.Unlock()
	}
	GetExecJob = func(cookie string) *execJob {
		execJobsManager.mu.RLock()
		job := execJobsManager.jobs[cookie]
		execJobsManager.mu.RUnlock()
		return job
	}
	RemoveExecJob = func(cookie string) {
		execJobsManager.mu.Lock()
		delete(execJobsManager.jobs, cookie)
		execJobsManager.mu.Unlock()
	}
)

// Export job type for testing
type ExecJob = execJob

// Export spawn function for testing
var SnapctlExecSpawn = snapctlExecSpawn
