// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package models

type Exportable struct {
	Sidecar Sidecar
	Genesis []byte
}
