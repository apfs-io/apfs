package v1

// IsUnknown response status
func (code ResponseStatusCode) IsUnknown() bool {
	return code == ResponseStatusCode_UNKNOWN_INVALID
}

// IsOK response status
func (code ResponseStatusCode) IsOK() bool {
	return code == ResponseStatusCode_OK
}

// IsFailed response status
func (code ResponseStatusCode) IsFailed() bool {
	return code == ResponseStatusCode_FAILED
}

// IsNotFound response status
func (code ResponseStatusCode) IsNotFound() bool {
	return code == ResponseStatusCode_NOT_FOUND
}
