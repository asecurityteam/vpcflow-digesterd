package errs

import "fmt"

// NotFound represents a resource lookup that failed due to a missing record.
type NotFound struct {
	ID string
}

func (e NotFound) Error() string {
	return fmt.Sprintf("resource %s was not found", e.ID)
}
