====================
 snapd-release-agent
====================

------------------------------------------------------
internal tool for observing vacancy of named v1 cgroup
------------------------------------------------------

:Author: zygmunt.krynicki@canonical.com
:Date:   2019-10-04
:Copyright: Canonical Ltd.
:Version: 2.42
:Manual section: 8
:Manual group: snappy

SYNOPSIS
========

	snapd-release-agent PATH

DESCRIPTION
===========

The `snapd-release-agent` is a program internally used by `snapd` to become
aware of the moment when a control group created by `snap-confine` becomes
empty, indicating that a given snap application or hook no longer has any
processes.

OPTIONS
=======

The `snapd-release-agent` program does not accept any options.

FEATURES
========

`snapd-release-agent` is invoked by automatically by the kernel when a cgroup,
constructed by `snap-confine`, for a particular snap application or hook, no
longer houses any processes. This is an indication that a given application or
hook is no longer running.

If, for whatever reason, the program is invoked in response to a non-snapd
cgroup, identified by the fact that the cgroup path does not start with
`snap.`, or if the cgroup path contains a directory separator, which means it
is a sub-hierarchy that is not directly constructed by `snap-confine` then the
program does simply exits without performing other operations.

The name of the cgroup is also the `$SNAP_SECURITY_TAG`. The tag has the
following form `snap.pkg-name.app-name` for applications and
`snap.pkg-name.hook.hook-name`, where `pkg-name` is the name of the snap
package or its instance (where a package is installed multiple times under
different names), `app-name` is the name of an application in the package and
`hook-name` is the name of one of the supported snapd hooks.

It is possible to derive `$SNAP_NAME` from `$SNAP_SECURITY_TAG`.

All remaining operations are performed under exclusive `flock(2)`-based lock
`/run/snapd/lock/$SNAP_NAME.lock`.

Removal of unused cgroups
-------------------------

The following unused cgroup hierarchies are removed:

 - The name=snapd cgroup hierarchy at `/run/snapd/cgroup/$SNAP_SECURITY_TAG`
 - The freezer cgroup hierarchy at `/sys/fs/cgroup/freezer/$SNAP_NAME`
 - The devices cgroup hierarchy at `sys/fs/cgroup/devices/$SNAP_SECURITY_TAG`

Removal of loaded apparmor profiles
-----------------------------------

If the snap is no longer installed on the system then apparmor profiles related
to that snap that were loaded into the kernel are also removed.

Removal of app or hook specific monitor file
--------------------------------------------

The release agent assists snapd with up-to-date information about liveliness of
any app or hook belonging to a given snap.

This is implemented through the following protocol.

1) The `snap-confine`, upon startup of a snap application or hook, creates a
file `/run/snapd/monitor/$SNAP_SECURITY_TAG`. The file may contain additional
information that is not interpreted by `snapd-release-agent` but may be of use
to `snapd`.

2) In addition, the started application or hook process is associated with a
`name=snapd` cgroup rooted at `/sys/fs/cgroup/snapd`, to track all the
processes belonging to that particular entry point.

3) The `snapd-release-agent` is invoked when all such processes cease to exist
and removes the monitor file.

This protocol operates irrespectively of the snapd service, even while the
service is inactive or restarting for upgrades.

Anyone interested in snap application life-cycle can establish a `inotify(2)`
or `fanotify(2)`-based event tracking on the single directory
`/run/snapd/monitor`.  This directory is guaranteed to be a `tmpfs` which
always supports event notification.

Anyone interested in reliably acting while a snap package, or a subset of that
package is not executing can acquire the correct lock, use the monitor files or
cgroup `cgroup.procs` file as source of further information. While the
*per-snap* lock is held `snap-confine` cannot start additional processes
associated with a given snap.

ENVIRONMENT
===========

`snapd-release-agent` does not use any environment variables.

FILES
=====

`snapd-release-agent` and `snap-confine` use the following files:

`/run/snapd/lock/$SNAP_NAME.lock`:

        Lock synchronizing access to per-snap monitor files and cgroup
        operations. This lock is acquired by `snap-confine`, `snap-update-ns`,
        `snapd-release-agent` and `snapd`.

`/run/snapd/monitor/$SNAP_SECURITY_TAG`:

        Monitor file for application liveness for a specific snap application
        or hook. The file is created by `snap-confine` and removed by
        `snapd-release-agent`.

`/run/snapd/cgroup`:

        Location of the name=snapd v1 cgroup used for tracking processes
        belonging to snap applications and hooks. This cgroup is mounted and
        configured by `snap-confine` to use this release agent. There are no
        controllers present here.

`/run/snapd/cgroup/$SNAP_SECURITY_TAG`:

        Cgroup v1 hierarchy for tracking processes belonging to a specific
        application or hook.

`/sys/fs/cgroup/devices/$SNAP_SECURITY_TAG`:

        Cgroup v1 hierarchy for controlling access to character and block
        devices. This cgroup is created indirectly by `snap-confine` and
        directly by `snapd-device-helper`.

        Due to bug https://bugs.launchpad.net/snapd/+bug/1838937 a process
        may not be associated with this cgroup even though it ought to.

`/sys/fs/cgroup/freezer/$SNAP_NAME`:

        Cgroup v1 hierarchy for freezing processes belonging to a specific
        mount namespace. This is a hardening feature to avoid races with
        untrusted application processes while mount operations are being
        performed.

        The freezer is not used when cgroup v2 is used in unified mode.
        Eventually new mount operations from Linux 5.2 will replace the need
        for freezing running apps.

`/sys/security/apparmor`:

        Kernel interface to the apparmor LSM. Specifically the `policy/profiles`
        directory is enumerated. The `name` attribute file of each profile is
        read and matching profiles are removed by writing to the `.remove`
        file.

`/snap/$SNAP_NAME` and `/var/lib/snapd/$SNAP_NAME`:

        Presence of one of those directories is equivalent with a snap being
        installed.

BUGS
====

Please report all bugs with https://bugs.launchpad.net/snapd/+filebug
