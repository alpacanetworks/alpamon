package pidfile

import "fmt"

func FilePath(name string) string {
	return fmt.Sprintf("/var/run/%s.pid", name)
}
