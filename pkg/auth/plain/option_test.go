package plain

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialsFromCSV(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output map[string]string
		err    error
	}{
		{
			name:   "empty input",
			input:  "",
			output: map[string]string{},
		},
		{
			name: "username,password ",
			input: `
				"alice","alice-secret"
				"bob",bob-secret
				charlie, "--b+t6e,8r4C>^M}"`,
			output: map[string]string{
				"alice":   "alice-secret",
				"bob":     "bob-secret",
				"charlie": "--b+t6e,8r4C>^M}",
			},
		},
		{
			name:   "malformed input",
			input:  `alice`,
			output: nil,
			err:    &csv.ParseError{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual, err := credentialsFromCSV(strings.NewReader(tc.input))
			a.Equal(tc.output, actual)
			a.Equal(tc.err != nil, err != nil, "expected errors differs")
			if tc.err != nil && err != nil {
				a.IsType(tc.err, err)
			}
		})
	}

}
