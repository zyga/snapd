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
#ifndef SNAP_CONFINE_SECCOMP_SUPPORT_EXT_H
#define SNAP_CONFINE_SECCOMP_SUPPORT_EXT_H

#include <linux/filter.h>
#include <stddef.h>

/**
 * sc_apply_seccomp_filter applies a given BPF program as a seccomp syscall
 * filter to the calling process.
 *
 * The function first tries the modern seccomp(2) syscall with
 * SECCOMP_FILTER_FLAG_LOG. If that fails (e.g., on older kernels), it falls
 * back to prctl(PR_SET_SECCOMP, ...).
 *
 * NO_NEW_PRIVS is intentionally not set because it interferes with AppArmor
 * exec transitions for certain snapd interfaces. The calling process should
 * already have appropriate capabilities (CAP_SYS_ADMIN) and be confined by an
 * AppArmor profile that blocks ptrace.
 **/
void sc_apply_seccomp_filter(struct sock_fprog *prog);

#endif
