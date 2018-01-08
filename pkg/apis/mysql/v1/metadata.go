package v1

import "github.com/oracle/mysql-operator/pkg/constants"

// SetOperatorVersionLabel sets the specified operator version label on the label map.
func SetOperatorVersionLabel(labelMap map[string]string, label string) {
	labelMap[constants.MySQLOperatorVersionLabel] = label
}

// GetOperatorVersionLabel get the specified operator version label on the label map.
func GetOperatorVersionLabel(labelMap map[string]string) string {
	return labelMap[constants.MySQLOperatorVersionLabel]
}
