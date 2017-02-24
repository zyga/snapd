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
#include "config.h"
#include "snap.h"

#include <regex.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

#include "../libsnap-confine-private/utils.h"
#include "../libsnap-confine-private/string-utils.h"

bool verify_security_tag(const char *security_tag)
{
	// The executable name is of form:
	// snap.<name>.(<appname>|hook.<hookname>)
	// - <name> must start with lowercase letter, then may contain
	//   lowercase alphanumerics and '-'
	// - <appname> may contain alphanumerics and '-'
	// - <hookname must start with a lowercase letter, then may
	//   contain lowercase letters and '-'
	const char *whitelist_re =
	    "^snap\\.[a-z](-?[a-z0-9])*\\.([a-zA-Z0-9](-?[a-zA-Z0-9])*|hook\\.[a-z](-?[a-z])*)$";
	regex_t re;
	if (regcomp(&re, whitelist_re, REG_EXTENDED | REG_NOSUB) != 0)
		die("can not compile regex %s", whitelist_re);

	int status = regexec(&re, security_tag, 0, NULL, 0);
	regfree(&re);

	return (status == 0);
}

void sc_snap_name_validate(const char *snap_name, struct sc_error **errorp)
{
	struct sc_error *err = NULL;

	const char *name_re = "^([a-z0-9]+-?)*[a-z](-?[a-z0-9])*$";

	regex_t re;
	if (regcomp(&re, name_re, REG_EXTENDED | REG_NOSUB) != 0) {
		die("cannot compile regex %s", name_re);
	}

	int status = regexec(&re, snap_name, 0, NULL, 0);
	regfree(&re);

	if (status != 0) {
		size_t quote_buf_size = strlen(snap_name) * 4 + 3;
		char *quote_buf = calloc(1, quote_buf_size);

		if (quote_buf == NULL) {
			err =
			    sc_error_init_from_errno(errno,
						     "cannot allocate memory for quoted name");
			goto out;
		}

		sc_string_quote(quote_buf, quote_buf_size, snap_name);
		err =
		    sc_error_init(SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME,
				  "invalid snap name %s", quote_buf);
		free(quote_buf);
	}

 out:
	sc_error_forward(errorp, err);
}
