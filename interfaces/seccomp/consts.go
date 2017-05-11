// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
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

package seccomp

import (
	"syscall"
)

// TIP: You can copy-paste a block of #define's from header files and use the
// following vim command to convert the selected block into proper golang code.
//
// :'<,'>s/#define\s\+\([A-Z0-9_]\+\)\s\+[0-9]\+.*/"\1": syscall.\1,/

// knownConstants maps between symbolic names and their integer values.
var knownConstants = map[string]int{
	// Socket address families.
	"AF_UNSPEC":     syscall.AF_UNSPEC,
	"AF_LOCAL":      syscall.AF_LOCAL,
	"AF_UNIX":       syscall.AF_UNIX,
	"AF_FILE":       syscall.AF_FILE,
	"AF_INET":       syscall.AF_INET,
	"AF_AX25":       syscall.AF_AX25,
	"AF_IPX":        syscall.AF_IPX,
	"AF_APPLETALK":  syscall.AF_APPLETALK,
	"AF_NETROM":     syscall.AF_NETROM,
	"AF_BRIDGE":     syscall.AF_BRIDGE,
	"AF_ATMPVC":     syscall.AF_ATMPVC,
	"AF_X25":        syscall.AF_X25,
	"AF_INET6":      syscall.AF_INET6,
	"AF_ROSE":       syscall.AF_ROSE,
	"AF_DECnet":     syscall.AF_DECnet,
	"AF_NETBEUI":    syscall.AF_NETBEUI,
	"AF_SECURITY":   syscall.AF_SECURITY,
	"AF_KEY":        syscall.AF_KEY,
	"AF_NETLINK":    syscall.AF_NETLINK,
	"AF_ROUTE":      syscall.AF_ROUTE,
	"AF_PACKET":     syscall.AF_PACKET,
	"AF_ASH":        syscall.AF_ASH,
	"AF_ECONET":     syscall.AF_ECONET,
	"AF_ATMSVC":     syscall.AF_ATMSVC,
	"AF_RDS":        syscall.AF_RDS,
	"AF_SNA":        syscall.AF_SNA,
	"AF_IRDA":       syscall.AF_IRDA,
	"AF_PPPOX":      syscall.AF_PPPOX,
	"AF_WANPIPE":    syscall.AF_WANPIPE,
	"AF_LLC":        syscall.AF_LLC,
	"AF_IB":         27, // syscall.AF_IB
	"AF_MPLS":       28, // syscall.AF_MPLS
	"AF_CAN":        syscall.AF_CAN,
	"AF_TIPC":       syscall.AF_TIPC,
	"AF_BLUETOOTH":  syscall.AF_BLUETOOTH,
	"AF_IUCV":       syscall.AF_IUCV,
	"AF_RXRPC":      syscall.AF_RXRPC,
	"AF_ISDN":       syscall.AF_ISDN,
	"AF_PHONET":     syscall.AF_PHONET,
	"AF_IEEE802154": syscall.AF_IEEE802154,
	"AF_CAIF":       syscall.AF_CAIF,
	"AF_ALG":        syscall.AF_ALG,
	"AF_NFC":        39, // syscall.AF_NFC
	"AF_VSOCK":      40, // syscall.AF_VSOCK

	// Socket types.
	"SOCK_STREAM": syscall.SOCK_STREAM,
	"SOCK_DGRAM":  syscall.SOCK_DGRAM,

	// Netlink
	"NETLINK_ROUTE":          syscall.NETLINK_ROUTE,
	"NETLINK_UNUSED":         syscall.NETLINK_UNUSED,
	"NETLINK_USERSOCK":       syscall.NETLINK_USERSOCK,
	"NETLINK_FIREWALL":       syscall.NETLINK_FIREWALL,
	"NETLINK_NFLOG":          syscall.NETLINK_NFLOG,
	"NETLINK_XFRM":           syscall.NETLINK_XFRM,
	"NETLINK_SELINUX":        syscall.NETLINK_SELINUX,
	"NETLINK_ISCSI":          syscall.NETLINK_ISCSI,
	"NETLINK_AUDIT":          syscall.NETLINK_AUDIT,
	"NETLINK_FIB_LOOKUP":     syscall.NETLINK_FIB_LOOKUP,
	"NETLINK_CONNECTOR":      syscall.NETLINK_CONNECTOR,
	"NETLINK_NETFILTER":      syscall.NETLINK_NETFILTER,
	"NETLINK_IP6_FW":         syscall.NETLINK_IP6_FW,
	"NETLINK_DNRTMSG":        syscall.NETLINK_DNRTMSG,
	"NETLINK_KOBJECT_UEVENT": syscall.NETLINK_KOBJECT_UEVENT,
	"NETLINK_GENERIC":        syscall.NETLINK_GENERIC,
	"NETLINK_SCSITRANSPORT":  syscall.NETLINK_SCSITRANSPORT,
	"NETLINK_ECRYPTFS":       syscall.NETLINK_ECRYPTFS,

	// Stuff from linux/stat.h
	"S_IFMT":   syscall.S_IFMT,
	"S_IFSOCK": syscall.S_IFSOCK,
	"S_IFLNK":  syscall.S_IFLNK,
	"S_IFREG":  syscall.S_IFREG,
	"S_IFBLK":  syscall.S_IFBLK,
	"S_IFDIR":  syscall.S_IFDIR,
	"S_IFCHR":  syscall.S_IFCHR,
	"S_IFIFO":  syscall.S_IFIFO,
	"S_ISUID":  syscall.S_ISUID,
	"S_ISGID":  syscall.S_ISGID,
	"S_ISVTX":  syscall.S_ISVTX,

	// Stuff from linux/resource.h
	"PRIO_PROCESS": syscall.PRIO_PROCESS,
	"PRIO_PGRP":    syscall.PRIO_PGRP,
	"PRIO_USER":    syscall.PRIO_USER,

	// Stuff from asm-generic/ioctls.h
	"TIOCSTI": syscall.TIOCSTI,
}
