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

// WorkRequestResource The representation of WorkRequestResource
type WorkRequestResource struct {

	// The status that a work request can present
	ActionType WorkRequestResourceActionTypeEnum `mandatory:"false" json:"actionType,omitempty"`

	// The resource type the work request is affects.
	EntityType *string `mandatory:"false" json:"entityType"`

	// The identifier of the resource the work request affects.
	Identifier *string `mandatory:"false" json:"identifier"`

	// The URI path that the user can do a GET on to access the resource metadata
	EntityUri *string `mandatory:"false" json:"entityUri"`

	// The metadata of the resource.
	Metadata map[string]string `mandatory:"false" json:"metadata"`
}

func (m WorkRequestResource) String() string {
	return common.PointerString(m)
}

// WorkRequestResourceActionTypeEnum Enum with underlying type: string
type WorkRequestResourceActionTypeEnum string

// Set of constants representing the allowable values for WorkRequestResourceActionTypeEnum
const (
	WorkRequestResourceActionTypeCreated    WorkRequestResourceActionTypeEnum = "CREATED"
	WorkRequestResourceActionTypeUpdated    WorkRequestResourceActionTypeEnum = "UPDATED"
	WorkRequestResourceActionTypeDeleted    WorkRequestResourceActionTypeEnum = "DELETED"
	WorkRequestResourceActionTypeRelated    WorkRequestResourceActionTypeEnum = "RELATED"
	WorkRequestResourceActionTypeInProgress WorkRequestResourceActionTypeEnum = "IN_PROGRESS"
	WorkRequestResourceActionTypeRead       WorkRequestResourceActionTypeEnum = "READ"
	WorkRequestResourceActionTypeWritten    WorkRequestResourceActionTypeEnum = "WRITTEN"
)

var mappingWorkRequestResourceActionType = map[string]WorkRequestResourceActionTypeEnum{
	"CREATED":     WorkRequestResourceActionTypeCreated,
	"UPDATED":     WorkRequestResourceActionTypeUpdated,
	"DELETED":     WorkRequestResourceActionTypeDeleted,
	"RELATED":     WorkRequestResourceActionTypeRelated,
	"IN_PROGRESS": WorkRequestResourceActionTypeInProgress,
	"READ":        WorkRequestResourceActionTypeRead,
	"WRITTEN":     WorkRequestResourceActionTypeWritten,
}

// GetWorkRequestResourceActionTypeEnumValues Enumerates the set of values for WorkRequestResourceActionTypeEnum
func GetWorkRequestResourceActionTypeEnumValues() []WorkRequestResourceActionTypeEnum {
	values := make([]WorkRequestResourceActionTypeEnum, 0)
	for _, v := range mappingWorkRequestResourceActionType {
		values = append(values, v)
	}
	return values
}
