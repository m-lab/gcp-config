package stctl

// These aliases allow unit tests in stctl_test package to access unexported items.
type JobMetadata = jobMetadata

var (
	ErrNotFound = errNotFound
	GetDesc     = getDesc
	Find        = (*Command).find
	SpecMatches = (*Command).specMatches
)
