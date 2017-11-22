#!/bin/bash

# shellcheck source=tests/lib/utils.sh
. "$TESTSLIB/utils.sh"

prepare_project() {
    # Check if running inside a container.
    # The testsuite will not work in such an environment
    if systemd-detect-virt -c; then
        echo "Tests cannot run inside a container"
        exit 1
    fi

    "$TESTSLIB"/prepare-project.sh
}

prepare_project_each() {
    # We want to rotate the logs so that when inspecting or dumping them we
    # will just see logs since the test has started.

    # Clear the systemd journal. Unfortunately the deputy-systemd on Ubuntu
    # 14.04 does not know about --rotate or --vacuum-time so we need to remove
    # the journal the hard way.
    case "$SPREAD_SYSTEM" in
        ubuntu-14.04-*)
            # Force a log rotation with small size
            sed -i.bak s/#SystemMaxUse=/SystemMaxUse=1K/g /etc/systemd/journald.conf
            systemctl kill --kill-who=main --signal=SIGUSR2 systemd-journald.service

            # Restore the initial configuration and rotate logs
            mv /etc/systemd/journald.conf.bak /etc/systemd/journald.conf
            systemctl kill --kill-who=main --signal=SIGUSR2 systemd-journald.service

            # Remove rotated journal logs
            systemctl stop systemd-journald.service
            find /run/log/journal/ -name "*@*.journal" -delete
            systemctl start systemd-journald.service
            ;;
        *)
            journalctl --rotate
            sleep .1
            journalctl --vacuum-time=1ms
            ;;
    esac

    # Clear the kernel ring buffer.
    dmesg -c > /dev/null
}

restore_project_each() {
    # Udev rules are notoriously hard to write and seemingly correct but subtly
    # wrong rules can pass review. Whenever that happens udev logs an error
    # message. As a last resort from lack of a better mechanism we can try to
    # pick up such errors.
    if grep "invalid .*snap.*.rules" /var/log/syslog; then
        echo "Invalid udev file detected, test most likely broke it"
        exit 1
    fi
}

restore_project() {
    # XXX: Why are we enabling autorefresh for external targets?
    if [ "$SPREAD_BACKEND" = external ] && [ -e /snap/core/current/meta/hooks/configure ]; then
        systemctl enable --now snapd.refresh.timer
        snap set core refresh.schedule=""
    fi

    # We use a trick to accelerate prepare/restore code in certain suites. That
    # code uses a tarball to store the vanilla state. Here we just remove this
    # tarball.
    rm -f "$SPREAD_PATH/snapd-state.tar.gz"

    # Remove all of the code we pushed and any build results. This removes
    # stale files and we cannot do incremental builds anyway so there's little
    # point in keeping them.
    if [ -n "$GOPATH" ]; then
        rm -rf "${GOPATH%%:*}"
    fi
}

prepare_suite() {
    # shellcheck source=tests/lib/prepare.sh
    . "$TESTSLIB/prepare.sh"
    if [[ "$SPREAD_SYSTEM" == ubuntu-core-16-* ]]; then
        prepare_all_snap
    else
        prepare_classic
    fi
}

prepare_suite_each() {
    "$TESTSLIB/reset.sh" --reuse-core
    # shellcheck source=tests/lib/prepare.sh
    . "$TESTSLIB/prepare.sh"
    if [[ "$SPREAD_SYSTEM" != ubuntu-core-16-* ]]; then
        prepare_each_classic
    fi
}

restore_suite_each() {
    true
}

restore_suite() {
    "$TESTSLIB/reset.sh" --store
    if [[ "$SPREAD_SYSTEM" != ubuntu-core-16-* ]]; then
        # shellcheck source=tests/lib/pkgdb.sh
        . "$TESTSLIB"/pkgdb.sh
        distro_purge_package snapd
        if [[ "$SPREAD_SYSTEM" != opensuse-* ]]; then
            # A snap-confine package never existed on openSUSE
            distro_purge_package snap-confine
        fi
    fi
}

case "$1" in
    --prepare-project)
        prepare_project
        ;;
    --prepare-project-each)
        prepare_project_each
        ;;
    --restore-project-each)
        restore_project_each
        ;;
    --restore-project)
        restore_project
        ;;
    --prepare-suite)
        prepare_suite
        ;;
    --prepare-suite-each)
        prepare_suite_each
        ;;
    --restore-suite-each)
        restore_suite_each
        ;;
    --restore-suite)
        restore_suite
        ;;
    *)
        echo "unsupported argument: $1"
        echo "try one of --{prepare,restore}-{project,suite}{,-each}"
        exit 1
        ;;
esac
