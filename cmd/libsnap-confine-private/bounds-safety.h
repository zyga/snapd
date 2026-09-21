/*
 * Copyright (C) 2026 Canonical Ltd
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

#ifndef SNAP_CONFINE_BOUNDS_SAFETY_H
#define SNAP_CONFINE_BOUNDS_SAFETY_H

#ifdef HAVE_CONFIG_H
#include "config.h"
#endif  // HAVE_CONFIG_H

/*
 * Portability layer for clang's -fbounds-safety bounds annotations.
 *
 * When the compiler implements the extension (a clang providing <ptrcheck.h>
 * and accepting -fbounds-safety) the real annotations from <ptrcheck.h> are
 * used and the compiler enforces bounds checking at run time.  On every other
 * toolchain (gcc, older clang) the annotations are macro-defined to empty, so
 * the source keeps compiling unchanged and the annotations are inert.
 *
 * See https://clang.llvm.org/docs/BoundsSafety.html for the semantics of each
 * annotation.  In short:
 *
 *   __single                 pointer to a single object (or NULL); the default
 *                            for ABI-visible pointers, no pointer arithmetic
 *   __counted_by(n)          pointer to n elements of the pointee type
 *   __counted_by_or_null(n)  as above, but the pointer may also be NULL
 *   __sized_by(n)            pointer to n bytes (for void * / incomplete types)
 *   __sized_by_or_null(n)    as above, but the pointer may also be NULL
 *   __ended_by(p)            pointer whose upper bound is the pointer p
 *   __ended_by_or_null(p)    as above, but the pointer may also be NULL
 *   __null_terminated        sentinel-terminated pointer (ends in NUL / 0)
 *   __terminated_by(t)       sentinel-terminated pointer (ends in value t)
 *   __indexable              wide pointer, indexable in the positive direction
 *   __bidi_indexable         wide pointer, indexable in both directions
 *   __unsafe_indexable       plain C pointer, no bounds information (interop)
 */
#if defined(__clang__) && defined(HAVE_BOUNDS_SAFETY) && defined(__has_include)
/* The configure-time HAVE_BOUNDS_SAFETY only means the toolchain supports the
 * extension; a given translation unit may still have it turned off (the unit
 * tests are built with -fno-bounds-safety).  Only use the real annotations when
 * the feature is actually enabled for this translation unit. */
#if defined(__has_feature)
#if (__has_feature(bounds_safety_attributes) || __has_feature(bounds_attributes)) && __has_include(<ptrcheck.h>)
#include <ptrcheck.h>
#define SC_HAVE_BOUNDS_SAFETY 1
#endif  // __has_feature && __has_include(<ptrcheck.h>)
#endif  // __has_feature
#endif  // __clang__ && HAVE_BOUNDS_SAFETY && __has_include

#ifndef SC_HAVE_BOUNDS_SAFETY
/* No bounds-safety-capable toolchain: make every annotation a no-op.  Undefine
 * any pre-existing definition first: some system headers (e.g. linux/stddef.h
 * pulled in via <fcntl.h>) define a few of these names for their own use. */
#undef __single
#define __single
#undef __counted_by
#define __counted_by(n)
#undef __counted_by_or_null
#define __counted_by_or_null(n)
#undef __sized_by
#define __sized_by(n)
#undef __sized_by_or_null
#define __sized_by_or_null(n)
#undef __ended_by
#define __ended_by(p)
#undef __ended_by_or_null
#define __ended_by_or_null(p)
#undef __null_terminated
#define __null_terminated
#undef __terminated_by
#define __terminated_by(t)
#undef __indexable
#define __indexable
#undef __bidi_indexable
#define __bidi_indexable
#undef __unsafe_indexable
#define __unsafe_indexable

/*
 * Interop conversion intrinsics.  On a bounds-safety toolchain these come from
 * <ptrcheck.h> and perform a (checked or unchecked) conversion between pointer
 * bounds annotations; elsewhere they are plain casts to the (annotation-free)
 * pointer type `T` so the code keeps compiling unchanged.  `T` is the full
 * pointer type, e.g. `__unsafe_forge_null_terminated(const char *, p)`.  The
 * operand must be a pointer; pass `&arr[0]` (not the array `arr`) when forging
 * a stack buffer.  Use them only at the boundary with code that has not adopted
 * bounds annotations (libc, glib, the kernel, ...).
 */
#define __unsafe_forge_single(T, P) ((T)(P))
#define __unsafe_forge_bidi_indexable(T, P, S) ((T)(P))
#define __unsafe_forge_terminated_by(T, P, E) ((T)(P))
#define __unsafe_forge_null_terminated(T, P) ((T)(P))
#define __terminated_by_to_indexable(P) (P)
#define __unsafe_terminated_by_to_indexable(P) (P)
#define __null_terminated_to_indexable(P) (P)
#define __unsafe_null_terminated_to_indexable(P) (P)
#define __unsafe_terminated_by_from_indexable(T, P) ((T)(P))
#define __unsafe_null_terminated_from_indexable(T, P) ((T)(P))
#endif  // SC_HAVE_BOUNDS_SAFETY

#endif  // SNAP_CONFINE_BOUNDS_SAFETY_H
