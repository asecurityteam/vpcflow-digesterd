package logs

const (
	// DependencyStorage identifies a storage failure
	DependencyStorage = "storage"

	//DependencyQueuer identifies a queuer failure
	DependencyQueuer = "queuer"

	// DependencyMarker identifies a marker failure
	DependencyMarker = "marker"
)

// DependencyFailure is logged when a downstream dependency fails
type DependencyFailure struct {
	Dependency string `logevent:"dependency"`
	Reason     string `xlogevent:"reason"`
	Message    string `xlogevent:"message,default=dependency-failure"`
}

// UnknownFailure is logged when an unexpected error occurs
type UnknownFailure struct {
	Reason  string `xlogevent:"reason"`
	Message string `xlogevent:"message,default=unknown-failure"`
}

// InvalidInput is logged when the input provided is not valid
type InvalidInput struct {
	Reason  string `xlogevent:"reason"`
	Message string `xlogevent:"message,default=invalid-input"`
}

// NotFound is logged when the requested resource is not found
type NotFound struct {
	Reason  string `xlogevent:"reason"`
	Message string `xlogevent:"message,default=not-found"`
}

// Conflict is logged when the input provided is not valid
type Conflict struct {
	Reason  string `xlogevent:"reason"`
	Message string `xlogevent:"message,default=conflict"`
}
