package runner

import (
	"fmt"
	"strconv"
	"syscall"
)

func (pc *PtyClient) setPtyCmdSysProcAttrAndEnv(uid, gid int, groupIds []string, env map[string]string) {
	pc.cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:    uint32(uid),
			Gid:    uint32(gid),
			Groups: convertGroupIds(groupIds),
		},
	}
	pc.cmd.Dir = env["HOME"]

	for key, value := range env {
		pc.cmd.Env = append(pc.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}

func convertGroupIds(groupIds []string) []uint32 {
	var gids []uint32
	for _, gidStr := range groupIds {
		gid, err := strconv.Atoi(gidStr)
		if err != nil {
			continue
		}
		gids = append(gids, uint32(gid))
	}
	return gids
}
