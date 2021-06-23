// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

// Package gnmi implements a gnmi server to mock a device with YANG models.
package synchronizerv2

import (
	"time"
)

type Synchronizer struct {
	outputFileName string
	postEnable     bool
	postTimeout    time.Duration
}