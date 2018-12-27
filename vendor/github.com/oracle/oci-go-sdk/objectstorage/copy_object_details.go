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

// CopyObjectDetails To use any of the API operations, you must be authorized in an IAM policy. If you're not authorized,
// talk to an administrator. If you're an administrator who needs to write policies to give users access, see
// Getting Started with Policies (https://docs.us-phoenix-1.oraclecloud.com/Content/Identity/Concepts/policygetstarted.htm).
type CopyObjectDetails struct {

	// The name of the object to be copied
	SourceObjectName *string `mandatory:"true" json:"sourceObjectName"`

	// The destination region object will be copied to. Please specify name of the region, for example "us-ashburn-1".
	DestinationRegion *string `mandatory:"true" json:"destinationRegion"`

	// The destination namespace object will be copied to.
	DestinationNamespace *string `mandatory:"true" json:"destinationNamespace"`

	// The destination bucket object will be copied to.
	DestinationBucket *string `mandatory:"true" json:"destinationBucket"`

	// The destination name for the copy object.
	DestinationObjectName *string `mandatory:"true" json:"destinationObjectName"`

	// The entity tag to match the target object.
	SourceObjectIfMatchETag *string `mandatory:"false" json:"sourceObjectIfMatchETag"`

	// The entity tag to match the target object.
	DestinationObjectIfMatchETag *string `mandatory:"false" json:"destinationObjectIfMatchETag"`

	// The entity tag to not match the target object.
	DestinationObjectIfNoneMatchETag *string `mandatory:"false" json:"destinationObjectIfNoneMatchETag"`

	// Arbitrary string keys and values for the user-defined metadata for the object. Keys must be in
	// "opc-meta-*" format. Avoid entering confidential information. If user enter value in this field, the value
	// will become the object metadata for destination Object. If no value pass in, the destination object will have
	// the exact object metadata as source object.
	DestinationObjectMetadata map[string]string `mandatory:"false" json:"destinationObjectMetadata"`
}

func (m CopyObjectDetails) String() string {
	return common.PointerString(m)
}
