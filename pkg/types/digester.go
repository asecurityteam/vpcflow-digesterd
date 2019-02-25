package types

import (
	"time"

	"github.com/asecurityteam/go-vpcflow"
)

// DigesterProvider takes a start and a stop time, and returns a digester bound by those times
type DigesterProvider func(start, stop time.Time) vpcflow.Digester
