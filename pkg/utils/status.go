package utils

import "strings"

// NormalizeStatus maps backend uppercase status values to lowercase contract values.
func NormalizeStatus(s string) string {
	switch strings.ToUpper(s) {
	case "SUCCESS":
		return "completed"
	case "FAILED":
		return "failed"
	case "PENDING":
		return "pending"
	default:
		return "processing"
	}
}
