package io

import "github.com/apfs-io/apfs/models"

// Status of the object processing
type Status = models.ObjectStatus

// Status list
const (
	StatusUndefined  = models.StatusUndefined
	StatusProcessing = models.StatusProcessing
	StatusOK         = models.StatusOK
	StatusError      = models.StatusError
	StatusNotFound   = models.StatusNotFound
)
