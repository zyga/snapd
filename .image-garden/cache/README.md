# Host <=> Guest Cache

This directory is efficiently shared between the host computer and guest
virtual machine while running spread integration tests using the image-garden
backend.

# snaps/

The snaps/ sub-directory may be created here to hold cache of all the snap
packages that were needed during test execution. The cache is automatically
reused across tests to greatly reduce the network traffic required to prepare
the system and execute tests.
