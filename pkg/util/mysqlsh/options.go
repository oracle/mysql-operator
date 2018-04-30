package mysqlsh

import (
	"fmt"
	"strings"
)

// Options holds the options passed to individual mysqlsh commands.
type Options map[string]string

// String encodes options as a Python dictionary string.
func (opts Options) String() string {
	vals := []string{}
	for k, v := range opts {
		vals = append(vals, fmt.Sprintf("'%s': %s", k, quoted(v)))
	}
	return fmt.Sprintf("{%s}", strings.Join(vals, ", "))
}

// quoted handles quoting string options vs. not quoting boolean options.
func quoted(s string) string {
	switch strings.ToLower(s) {
	case "true":
		return "True"
	case "false":
		return "False"
	default:
		return fmt.Sprintf("'%s'", s)
	}
}
