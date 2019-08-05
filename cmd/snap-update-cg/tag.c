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

#include "config.h"

#include "tag.h"

#include <string.h>

#include "../libsnap-confine-private/string-utils.h"

char *snap_security_tag_to_udev_tag(const char *security_tag) {
    char *udev_tag = sc_strdup(security_tag);
    for (char *c = strchr(udev_tag, '.'); c != NULL; c = strchr(c, '.')) {
        *c = '_';
    }
    return udev_tag;
}
