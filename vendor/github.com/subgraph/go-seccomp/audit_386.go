// Copyright 2015 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package seccomp

// #include <linux/audit.h>
import "C"

const (
	auditArch = C.AUDIT_ARCH_I386
	nbits     = 32
)
