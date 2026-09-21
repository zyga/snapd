/*
 * Copyright (C) 2016-2017 Canonical Ltd
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

#include "string-utils.h"

#include <errno.h>
#include <stdarg.h>
#include <stdio.h>
#include <string.h>

#include "utils.h"

bool sc_streq(const char *__null_terminated a, const char *__null_terminated b) {
    if (!a || !b) {
        return false;
    }

    return strcmp(a, b) == 0;
}

bool sc_endswith(const char *__null_terminated str, const char *__null_terminated suffix) {
    if (!str || !suffix) {
        return false;
    }

    size_t xlen = strlen(suffix);
    size_t slen = strlen(str);

    if (slen < xlen) {
        return false;
    }

    return strncmp(__null_terminated_to_indexable(str) - xlen + slen, suffix, xlen) == 0;
}

bool sc_startswith(const char *__null_terminated str, const char *__null_terminated prefix) {
    if (!str || !prefix) {
        return false;
    }

    size_t xlen = strlen(prefix);
    return strncmp(str, prefix, xlen) == 0;
}

char *__null_terminated sc_strdup(const char *__null_terminated str) {
    // Set errno in case we die.
    errno = 0;
    size_t len;
    char *copy;
    if (str == NULL) {
        die("cannot duplicate NULL string");
    }
    len = strlen(str);
    copy = malloc(len + 1);
    if (copy == NULL) {
        die("cannot allocate string copy (len: %zd)", len);
    }
    /* We copy len+1 bytes to include the terminating NUL.  The checked
     * __null_terminated_to_indexable() conversion yields a bound that excludes
     * the terminator, so copying len+1 bytes through it would trap.  We know
     * str is NUL-terminated, so use the unchecked conversion here. */
    memcpy(copy, __unsafe_null_terminated_to_indexable(str), len + 1);
    return __unsafe_forge_null_terminated(char *, copy);
}

int sc_must_snprintf(char *__counted_by(size) str, size_t size, const char *__null_terminated format, ...) {
    // Set errno in case we die.
    errno = 0;
    int n;

    va_list va;
    va_start(va, format);
    n = vsnprintf(str, size, format, va);
    va_end(va);

    if (n < 0 || (size_t)n >= size) die("cannot format string: %s", str);

    return n;
}

size_t sc_string_append(char *__counted_by(dst_size) dst, size_t dst_size, const char *__null_terminated str) {
    // Set errno in case we die.
    errno = 0;
    if (dst == NULL) {
        die("cannot append string: buffer is NULL");
    }
    if (str == NULL) {
        die("cannot append string: string is NULL");
    }
    size_t dst_len = strnlen(dst, dst_size);
    if (dst_len == dst_size) {
        die("cannot append string: dst is unterminated");
    }

    size_t max_str_len = dst_size - dst_len;
    size_t str_len = strnlen(str, max_str_len);
    if (str_len == max_str_len) {
        die("cannot append string: str is too long or unterminated");
    }
    // Append the string
    memcpy(dst + dst_len, __null_terminated_to_indexable(str), str_len);
    // Ensure we are terminated
    dst[dst_len + str_len] = '\0';
    // return the new size
    return strlen(dst);
}

size_t sc_string_append_char(char *__counted_by(dst_size) dst, size_t dst_size, char c) {
    // Set errno in case we die.
    errno = 0;
    if (dst == NULL) {
        die("cannot append character: buffer is NULL");
    }
    size_t dst_len = strnlen(dst, dst_size);
    if (dst_len == dst_size) {
        die("cannot append character: dst is unterminated");
    }
    size_t max_str_len = dst_size - dst_len;
    if (max_str_len < 2) {
        die("cannot append character: not enough space");
    }
    if (c == 0) {
        die("cannot append character: cannot append string terminator");
    }
    // Append the character and terminate the string.
    dst[dst_len + 0] = c;
    dst[dst_len + 1] = '\0';
    // Return the new size
    return dst_len + 1;
}

size_t sc_string_append_char_pair(char *__counted_by(dst_size) dst, size_t dst_size, char c1, char c2) {
    // Set errno in case we die.
    errno = 0;
    if (dst == NULL) {
        die("cannot append character pair: buffer is NULL");
    }
    size_t dst_len = strnlen(dst, dst_size);
    if (dst_len == dst_size) {
        die("cannot append character pair: dst is unterminated");
    }
    size_t max_str_len = dst_size - dst_len;
    if (max_str_len < 3) {
        die("cannot append character pair: not enough space");
    }
    if (c1 == 0 || c2 == 0) {
        die("cannot append character pair: cannot append string terminator");
    }
    // Append the two characters and terminate the string.
    dst[dst_len + 0] = c1;
    dst[dst_len + 1] = c2;
    dst[dst_len + 2] = '\0';
    // Return the new size
    return dst_len + 2;
}

void sc_string_init(char *__counted_by(buf_size) buf, size_t buf_size) {
    errno = 0;
    if (buf == NULL) {
        die("cannot initialize string, buffer is NULL");
    }
    if (buf_size == 0) {
        die("cannot initialize string, buffer is too small");
    }
    buf[0] = '\0';
}

void sc_string_quote(char *__counted_by(buf_size) buf, size_t buf_size, const char *__null_terminated str) {
    // Set errno in case we die.
    errno = 0;
    if (str == NULL) {
        die("cannot quote string: string is NULL");
    }
    const char *hex = "0123456789abcdef";
    // NOTE: this also checks buf/buf_size sanity so that we don't have to.
    sc_string_init(buf, buf_size);
    sc_string_append_char(buf, buf_size, '"');
    for (unsigned char c; (c = *str) != 0; ++str) {
        switch (c) {
                // Pass ASCII letters and digits unmodified.
            case '0' ... '9':
            case 'A' ... 'Z':
            case 'a' ... 'z':
                // Pass most of the punctuation unmodified.
            case ' ':
            case '!':
            case '#':
            case '$':
            case '%':
            case '&':
            case '(':
            case ')':
            case '*':
            case '+':
            case ',':
            case '-':
            case '.':
            case '/':
            case ':':
            case ';':
            case '<':
            case '=':
            case '>':
            case '?':
            case '@':
            case '[':
            case '\'':
            case ']':
            case '^':
            case '_':
            case '`':
            case '{':
            case '|':
            case '}':
            case '~':
                sc_string_append_char(buf, buf_size, c);
                break;
                // Escape special whitespace characters.
            case '\n':
                sc_string_append_char_pair(buf, buf_size, '\\', 'n');
                break;
            case '\r':
                sc_string_append_char_pair(buf, buf_size, '\\', 'r');
                break;
            case '\t':
                sc_string_append_char_pair(buf, buf_size, '\\', 't');
                break;
            case '\v':
                sc_string_append_char_pair(buf, buf_size, '\\', 'v');
                break;
                // Escape the escape character.
            case '\\':
                sc_string_append_char_pair(buf, buf_size, '\\', '\\');
                break;
                // Escape double quote character.
            case '"':
                sc_string_append_char_pair(buf, buf_size, '\\', '"');
                break;
                // Escape everything else as a generic hexadecimal escape string.
            default:
                sc_string_append_char_pair(buf, buf_size, '\\', 'x');
                sc_string_append_char_pair(buf, buf_size, hex[c >> 4], hex[c & 15]);
                break;
        }
    }
    sc_string_append_char(buf, buf_size, '"');
}

void sc_string_split(const char *__null_terminated string, char delimiter,
                     char *__counted_by_or_null(prefix_size) prefix_buf, size_t prefix_size,
                     char *__counted_by_or_null(suffix_size) suffix_buf, size_t suffix_size) {
    if (string == NULL) {
        die("internal error: cannot split string when it is unset");
    }
    if (prefix_buf == NULL && suffix_buf == NULL) {
        die("internal error: cannot split string when both prefix and suffix are unset");
    }

    const char *__null_terminated pos =
        __unsafe_forge_null_terminated(const char *, strchr(__null_terminated_to_indexable(string), delimiter));
    const char *__null_terminated suffix_start = "";
    size_t prefix_len = 0;
    size_t suffix_len = 0;
    if (pos == NULL) {
        prefix_len = strlen(string);
    } else {
        prefix_len = pos - string;
        suffix_start = pos + 1;
        suffix_len = strlen(suffix_start);
    }

    if (prefix_buf != NULL) {
        if (prefix_len >= prefix_size) {
            die("prefix buffer too small");
        }

        memcpy(prefix_buf, __null_terminated_to_indexable(string), prefix_len);
        prefix_buf[prefix_len] = '\0';
    }

    if (suffix_buf != NULL) {
        if (suffix_len >= suffix_size) {
            die("suffix buffer too small");
        }
        memcpy(suffix_buf, __null_terminated_to_indexable(suffix_start), suffix_len);
        suffix_buf[suffix_len] = '\0';
    }
}

char *__null_terminated sc_str_chomp(char *__null_terminated string) {
    size_t len = strlen(string);
    size_t pos = len;
    char *__bidi_indexable indexed = __null_terminated_to_indexable(string);
    for (; pos > 0 && indexed[pos - 1] == '\n'; --pos);
    if (pos < len) {
        indexed[pos] = '\0';
    }

    return string;
}
