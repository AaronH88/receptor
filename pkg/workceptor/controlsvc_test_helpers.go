//go:build !no_workceptor
// +build !no_workceptor

package workceptor

// HelperStrFromMap is a test helper function that exposes the strFromMap function for testing
func HelperStrFromMap(config map[string]interface{}, name string) (string, error) {
	return strFromMap(config, name)
}

// HelperIntFromMap is a test helper function that exposes the intFromMap function for testing
func HelperIntFromMap(config map[string]interface{}, name string) (int64, error) {
	return intFromMap(config, name)
}

// HelperBoolFromMap is a test helper function that exposes the boolFromMap function for testing
func HelperBoolFromMap(config map[string]interface{}, name string) (bool, error) {
	return boolFromMap(config, name)
}

// TestProcessSignature is a test helper function that exposes the processSignature function for testing
func (c *workceptorCommand) TestProcessSignature(workType, signature string, connIsUnix, signWork bool) error {
	return c.processSignature(workType, signature, connIsUnix, signWork)
}

// HelperGetSignWorkFromStatus is a test helper function that exposes the getSignWorkFromStatus function for testing
func HelperGetSignWorkFromStatus(status *StatusFileData) bool {
	return getSignWorkFromStatus(status)
}
