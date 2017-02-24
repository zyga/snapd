/*
 * Copyright (C) 2015 Canonical Ltd
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

#include <stdio.h>

#include "../libsnap-confine-private/snap.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"
#include "../snap-confine/ns-support.h"

int main(int argc, char **argv)
{
	if (argc != 2) {
		die("Usage: %s [--all | snap-name]", argv[0]);
	}
	// Create a new namespace manager so that it can handle locking and
	// enumeration for us. We want to die on any error so we're not providing
	// an outgoing error pointer.
	struct sc_ns_manager *mgr = sc_ns_manager_new(NULL);

	// Lock the entire namespace control directory.
	//
	// When snap confine runs it always takes this lock to check if it needs to
	// set up special sharing permissions and so we can lock out all instances
	// of snap-confine or snap-discard-ns that concurrently try to access or
	// modify the any namespace.
	//
	// NOTE: we could capture the error here and match against the errno domain
	// and EINTR and provide a dedicated error message but so far this is not
	// required.
	sc_ns_manager_lock_all(mgr, NULL);

	// With the lock held look at the command line argument. If it is "--all" then
	// enumrerate all the namespaces and discard them one by one. If anything but
	// that just assume it is a namespace name and discard it.
	if (sc_streq(argv[1], "--all")) {
		char **ns_names = sc_ns_manager_enumerate_ns_names(mgr, NULL);
		for (char **ns_name = ns_names; *ns_name != NULL; ns_name++) {
			struct sc_error *err = NULL;
			sc_ns_manager_discard_ns(mgr, *ns_name, &err);
			// If we cannot discard the namespace because the name is funky
			// let's log the problem but continue. We will discard as many
			// namespaces as we can.
			if (sc_error_match
			    (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME)) {
				fprintf(stderr, "(ignored) %s\n",
					sc_error_msg(err));
				sc_error_free(err);
				err = NULL;
			}
			free(*ns_name);
		}
		free(ns_names);
	} else {
		sc_ns_manager_discard_ns(mgr, argv[1], NULL);
	}

	// Unlock and destroy the manager when done. Technically we don't need to
	// unlock but this just feels cleaner.
	sc_ns_manager_unlock_all(mgr, NULL);
	sc_ns_manager_destroy(mgr);
	return 0;
}
