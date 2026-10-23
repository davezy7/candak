package models

import "errors"

// Common domain errors.
var (
	ErrInvalidURL       = errors.New("invalid or empty download URL")
	ErrDownloadNotFound = errors.New("download not found")
	ErrAlreadyRunning   = errors.New("download is already actively running")
	ErrAlreadyCompleted = errors.New("download is already completed")
	ErrDestinationAccess= errors.New("failed to access or create destination file")
)
