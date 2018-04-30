package mysqlsh

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsToString(t *testing.T) {
	testCases := []struct {
		name string
		in   Options
		out  string
	}{
		{
			name: "string_only",
			in: Options{
				"memberSslMode": "DISABLED",
				"ipWhitelist":   "10.0.0.0/8",
			},
			out: "{'memberSslMode': 'DISABLED', 'ipWhitelist': '10.0.0.0/8'}",
		}, {
			name: "with_bool",
			in: Options{
				"memberSslMode": "DISABLED",
				"ipWhitelist":   "10.0.0.0/8",
				"force":         "True",
				"multiMaster":   "True",
			},
			out: "{'memberSslMode': 'DISABLED', 'ipWhitelist': '10.0.0.0/8', 'force': True, 'multiMaster': True}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parts := strings.Split(strings.Trim(tc.out, "{}"), ", ")
			assert.Len(t, parts, len(tc.in))
			for k, v := range tc.in {
				assert.Contains(t, parts, fmt.Sprintf("'%s': %s", k, quoted(v)))
			}
		})
	}
}
