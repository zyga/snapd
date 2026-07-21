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
#ifndef SNAP_CONFINE_SELINUX_SUPPORT_H
#define SNAP_CONFINE_SELINUX_SUPPORT_H

/**
 * sc_selinux_set_snap_execcon sets up SELinux context transition for the snap.
 *
 * When the current process is running under the snappy_confine_t domain (as
 * entered via snap-confine's AppArmor profile on SELinux systems), this
 * function configures setexeccon() to transition to unconfined_service_t on
 * the next exec() call. This allows the actual workload binary to run without
 * being constrained by snap-confine's SELinux domain, which has no full policy
 * coverage for services or hooks running inside snaps.
 *
 * Returns 0 on success. If SELinux is not enabled, returns 0 as a no-op.
 * On error, calls die() with a descriptive message.
 *
 * Available only when compiled with HAVE_SELINUX (requires --with-selinux).
 **/
int sc_selinux_set_snap_execcon(void);

#endif /* SNAP_CONFINE_SELINUX_SUPPORT_H */
