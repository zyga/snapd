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

#include <linux/limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libsnap-confine-private/string-utils.h"

#include "config.h"

#include "../libsnap-confine-private/bounds-safety.h"

// Systemd environment generators work since version 233 which ships
// in Ubuntu 17.10+
int main(int argc, char *__null_terminated *__counted_by(argc) argv) {
    const char *__null_terminated snap_bin_dir = STATIC_SNAP_MOUNT_DIR "/bin";

    char *__unsafe_indexable path = getenv("PATH");
    if (path == NULL || sc_streq(__unsafe_forge_null_terminated(char *, path), "")) {
        // do nothing, until systemd is fixed, see LP#1791691
        return 0;
    }
    char buf[PATH_MAX + 1] = {0};
    strncpy(buf, path, sizeof(buf) - 1);
    char *__unsafe_indexable s = buf;

    char *__unsafe_indexable tok = strsep(&s, ":");
    while (tok != NULL) {
        if (sc_streq(__unsafe_forge_null_terminated(char *, tok), snap_bin_dir)) return 0;
        tok = strsep(&s, ":");
    }

    printf("PATH=%s:%s\n", path, snap_bin_dir);
    return 0;
}
