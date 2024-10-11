package runner

import "fmt"

// setFtpCmdSysProcAttrAndEnv does not set SysProcAttr on macOS because macOS does not support it.
func (fc *FtpClient) setFtpCmdSysProcAttrAndEnv(uid, gid int, groupIds []string, env map[string]string) {
	fc.cmd.Dir = env["HOME"]

	for key, value := range env {
		fc.cmd.Env = append(fc.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}
