/*
 * Copyright (C) 2016 Canonical Ltd
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

#ifndef SNAP_NAMESPACE_SUPPORT
#define SNAP_NAMESPACE_SUPPORT

#include <stdbool.h>

#include "../libsnap-confine-private/error.h"

#include "apparmor-support.h"

/**
 * Initialize namespace sharing.
 *
 * This function must be called once in each process that wishes to create or
 * join a namespace group.
 *
 * It is responsible for bind mounting the control directory over itself and
 * making it private (unsharing it with all the other peers) so that it can be
 * used for storing preserved namespaces as bind-mounted files from the nsfs
 * filesystem (namespace filesystem).
 *
 * This function acquires a flock(2)-based lock to ensure that no other instance
 * of snap-confine attempts to do this concurrently. If a process dies for any
 * reason then the lock is released and other instances of snap-confine can
 * complete the initialization.
 *
 * This function inspects /proc/self/mountinfo to determine if the directory
 * where namespaces are kept (/run/snapd/ns) is correctly prepared as described
 * above.
 *
 * For more details see namespaces(7).
 **/
void sc_initialize_ns_groups();

/**
 * Data required to manage namespaces amongst a group of processes.
 */
struct sc_ns_group;

enum {
	SC_NS_FAIL_GRACEFULLY = 1
};

/**
 * Open a namespace group.
 *
 * This will open and keep file descriptors for /run/snapd/ns/ as well as for
 * /run/snapd/ns/${group_name}.lock. The lock file is created if necessary but
 * is not locked until sc_lock_ns_mutex() is called.
 *
 * If the flags argument is SC_NS_FAIL_GRACEFULLY then the function returns
 * NULL if the /run/snapd/ns directory doesn't exist. In all other cases it
 * calls die() and exits the process.
 */
struct sc_ns_group *sc_open_ns_group(const char *group_name,
				     const unsigned flags);

/**
 * Close namespace group.
 *
 * This will close all of the open file descriptors and release allocated memory.
 */
void sc_close_ns_group(struct sc_ns_group *group);

/**
 * Acquire exclusive lock to the namespace group.
 *
 * This will attempt to acquire an flock-based exclusive lock on the file
 * descriptor associated with /run/snapd/ns/${group_name}.lock. If the process
 * is killed while the lock is held the lock is automatically released by the
 * kernel.
 *
 * The following methods should be called only while holding the lock:
 * - sc_create_or_join_ns_group()
 * - sc_should_populate_ns_group()
 * - sc_preserve_populated_ns_group()
 * - sc_discard_preserved_ns_group()
 **/
void sc_lock_ns_mutex(struct sc_ns_group *group);

/**
 * Release lock to the namespace group.
 *
 * This will attempt to release a flock-based lock on the file descriptor
 * associated with /run/snapd/ns/${group_name}.lock.
 **/
void sc_unlock_ns_mutex(struct sc_ns_group *group);

/**
 * Join the mount namespace associated with this group if one exists.
 *
 * Technically the function opens /run/snapd/ns/${group_name}.mnt and tries to
 * use setns() with the obtained file descriptor. If the call succeeds then the
 * function returns and subsequent call to sc_should_populate_ns_group() will
 * return false.
 *
 * If the call fails then an eventfd is constructed and a support process is
 * forked. The child process waits until data is written to the eventfd (this
 * can be done by calling sc_preserve_populated_ns_group()). In the meantime
 * the parent process unshares the mount namespace and sets a flag so that
 * sc_should_populate_ns_group() returns true.
 *
 * @returns true if the mount namespace needs to be populated
 **/
void sc_create_or_join_ns_group(struct sc_ns_group *group,
				struct sc_apparmor *apparmor);

/**
 * Check if the namespace needs to be populated.
 *
 * If the return value is true then at this stage the namespace is already
 * unshared. The caller should perform any mount operations that are desired
 * and then proceed to call sc_preserve_populated_ns_group().
 **/
bool sc_should_populate_ns_group(struct sc_ns_group *group);

/**
 * Preserve prepared namespace group.
 *
 * This function signals the child support process for namespace capture to
 * perform the capture and shut down. It must be called after the call to
 * sc_create_or_join_ns_group() and only when sc_should_populate_ns_group()
 * returns true.
 *
 * Technically this function writes to an eventfd that causes the child process
 * to wake up, bind mount /proc/$ppid/ns/mnt to /run/snapd/ns/${group_name}.mnt
 * and then exit. The parent process (the caller) then collects the child
 * process and returns.
 **/
void sc_preserve_populated_ns_group(struct sc_ns_group *group);

/**
 * Discard the preserved namespace group.
 *
 * This function unmounts the bind-mounted files representing the kernel mount
 * namespace.
 **/
void sc_discard_preserved_ns_group(struct sc_ns_group *group);

// -- namespace manager API

/**
 * Data required by the namespace manager.
 **/
struct sc_ns_manager;

/**
 * Allocate and initialize a namespace manager.
 *
 * Internally this will create (if required) and open /run/snapd/ns and
 * /run/snapd/ns/.lock.
 *
 * The error protocol is observed so if the caller doesn't provide an outgoing
 * error pointer the function will die on any error.
 *
 * In the case of an error there is no cleanup that needs to happen and the
 * function always returns NULL.
 *
 * When successful the caller should destroy the object when no longer
 * required. As a general rule, none of the resources allocated by this
 * function leak to children processes.
 **/
struct sc_ns_manager *sc_ns_manager_new(struct sc_error **errorp);

/**
 * De-initialize and deallocate a namespace manager.
 *
 * It is always safe to call with a NULL pointer.
 **/
void sc_ns_manager_destroy(struct sc_ns_manager *mgr);

/**
 * Lock the master lock of namespace control directory.
 *
 * This lock is taken by all the tools that access the data in the control
 * directory. The lock is automatically released if the process dies.
 *
 * The error protocol is observed so if the caller doesn't provide an outgoing
 * error pointer the function will die on any error.
 *
 * This is a blocking operation but all the programs in snapd take special care
 * to hold it only very briefly. As a sanity check this function waits for at
 * most a few seconds (implementation defined) for the lock to become
 * available.
 *
 * Programs using this function must make sure not to keep the lock held for
 * unbound amount of time.
 **/
void sc_ns_manager_lock_all(struct sc_ns_manager *mgr,
			    struct sc_error **errorp);

/**
 * Unlock the master lock of namespace control directory.
 *
 * The error protocol is observed so if the caller doesn't provide an outgoing
 * error pointer the function will die on any error.
 **/
void sc_ns_manager_unlock_all(struct sc_ns_manager *mgr,
			      struct sc_error **errorp);

/**
 * Enumerate all the namespaces in the control directory.
 *
 * The return value is a newly allocated array of strings terminated with a
 * NULL pointer. The caller is responsible for releasing each element as well
 * as the whole array. Unless an error is occurred the result is never NULL.
 * 
 * Each element of the array is simply a snap name as namespaces are named
 * after the snap they belong to.
 **/
char **sc_ns_manager_enumerate_ns_names(struct sc_ns_manager *mgr,
					struct sc_error **errorp);

/**
 * Discard a preserved namespace.
 *
 * The given namespace is discarded. Note that it is OK to call this on a namespace
 * that does not exist or was already unmounted (it is not an error).
 *
 * The error protocol is observed so if the caller doesn't provide an outgoing
 * error pointer the function will die on any error.
 **/
void sc_ns_manager_discard_ns(struct sc_ns_manager *mgr, const char *ns_name,
			      struct sc_error **errorp);

#endif
