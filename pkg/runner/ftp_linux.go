package runner

import (
	"fmt"
	"syscall"

	"github.com/alpacanetworks/alpamon-go/pkg/utils"
)

func (fc *FtpClient) setFtpCmdSysProcAttrAndEnv(uid, gid int, groupIds []string, env map[string]string) {
	fc.cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:    uint32(uid),
			Gid:    uint32(gid),
			Groups: utils.ConvertGroupIds(groupIds),
		},
	}
	fc.cmd.Dir = env["HOME"]

	for key, value := range env {
		fc.cmd.Env = append(fc.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}
