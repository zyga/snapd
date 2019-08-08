#!/bin/sh -ex
echo "Install a test service"
snap pack test-snapd-service
snap install --dangerous ./test-snapd-service_1.0_all.snap

echo "Extract the PID of the main process tracked by systemd"
# It would be nicer to use "systemctl show --property=... --value" but it doesn't work on older systemd.
pid=$(systemctl show snap.test-snapd-service.test-snapd-service.service --property=ExecMainPID | cut -d = -f 2)

echo "Extract the device cgroup of the main process"
effective_device_cgroup=$(grep devices < "/proc/$pid/cgroup" | cut -d : -f 3)
test "$effective_device_cgroup" = /snap.test-snapd-service.test-snapd-service

echo "Ensure that the claim of the process is consistent with the claim of the cgroup"
# This is just a sanity check.
MATCH "$pid" < "/sys/fs/cgroup/devices/$effective_device_cgroup/cgroup.procs"

echo "Verify the constraints imposed by the device cgroup made by snapd"
# This may change over time as it is governed by systemd.
test 'a *:* rwm' = "$(cat "/sys/fs/cgroup/devices/$effective_device_cgroup/devices.list")"

echo "Connect the joystick interface"
snap connect test-snapd-service:joystick

echo "Refresh the value of the main pid and the effective device cgroup after snap connect"
# NOTE: As of snapd 2.40 the PID and cgroup are expected to be the same as before.
pid=$(systemctl show snap.test-snapd-service.test-snapd-service.service --property=ExecMainPID | cut -d = -f 2)
effective_device_cgroup=$(grep devices < "/proc/$pid/cgroup" | cut -d : -f 3)

echo "Verify that the main process is still in the snapd-made cgroup"
test "$effective_device_cgroup" = /snap.test-snapd-service.test-snapd-service

echo "Verify the constraints imposed by the device cgroup made by snapd"
# NOTE: the actual permissions may drift over time. We just care about the fact
# that there *are* some constraints here now and there were none before.
test 'c 1:3 rwm' = "$(head -n 1 "/sys/fs/cgroup/devices/snap.test-snapd-service.test-snapd-service/devices.list")"

# TODO: disconnect and re-verify that we have "a *:* rwm"
