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

// ObjectNameFilter A filter that compares object names to a set of object name prefixes to determine if a rule applies to a
// given object.
type ObjectNameFilter struct {

	// An array of object name prefixes that the rule will apply to. An empty array means to include all objects.
	InclusionPrefixes []string `mandatory:"false" json:"inclusionPrefixes"`
}

func (m ObjectNameFilter) String() string {
	return common.PointerString(m)
}
