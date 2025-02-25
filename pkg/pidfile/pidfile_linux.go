package pidfile

func FilePath(name string) string {
	return fmt.Sprintf("/var/run/%s", name)
}
