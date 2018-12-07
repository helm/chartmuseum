// Copyright (c) 2016, 2018, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

// Object Storage Service API
//
// The Object and Archive Storage APIs for managing buckets and objects.
//

package objectstorage

import (
	"github.com/oracle/oci-go-sdk/common"
)

// WorkRequestLogEntry The representation of WorkRequestLogEntry
type WorkRequestLogEntry struct {

	// Human-readable log message.
	Message *string `mandatory:"false" json:"message"`

	// The time the log message was written. An RFC3339 formatted datetime string
	Timestamp *common.SDKTime `mandatory:"false" json:"timestamp"`
}

func (m WorkRequestLogEntry) String() string {
	return common.PointerString(m)
}
