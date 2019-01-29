package digesterd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterSlice(t *testing.T) {
	tc := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{
			Name:   "empty_slice",
			Input:  []string{},
			Output: []string{},
		},
		{
			Name:   "filtered_slice",
			Input:  []string{"", "a", "", "b", "", "c", ""},
			Output: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Output, filterSlice(tt.Input))
		})
	}
}

func TestMakePrefix(t *testing.T) {
	tc := []struct {
		Name     string
		Regions  []string
		Accounts []string
		Date     time.Time
		Prefix   string
	}{
		{
			Name:     "no_region",
			Regions:  []string{},
			Accounts: []string{"a"},
			Date:     time.Now(),
			Prefix:   "",
		},
		{
			Name:     "no_account",
			Regions:  []string{"r"},
			Accounts: []string{},
			Date:     time.Now(),
			Prefix:   "",
		},
		{
			Name:     "single_digits",
			Regions:  []string{"r"},
			Accounts: []string{"a"},
			Date:     time.Date(2019, time.January, 1, 0, 0, 0, 0, time.Local),
			Prefix:   "AWSLogs/a/vpcflowlogs/r/2019/01/01",
		},
		{
			Name:     "double_digits",
			Regions:  []string{"r"},
			Accounts: []string{"a"},
			Date:     time.Date(2019, time.October, 12, 0, 0, 0, 0, time.Local),
			Prefix:   "AWSLogs/a/vpcflowlogs/r/2019/10/12",
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Prefix, makePrefix(tt.Regions, tt.Accounts, tt.Date))
		})
	}
}
