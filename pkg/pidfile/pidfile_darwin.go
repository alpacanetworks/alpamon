package pidfile

import "fmt"

// Use /tmp for testing on macOS.
func FilePath(name string) string {
	return fmt.Sprintf("/tmp/%s.pid", name)
}
