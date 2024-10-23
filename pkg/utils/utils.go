package utils

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/host"
)

var (
	PlatformLike string
)

func InitPlatform() {
	getPlatformLike()
}

func getPlatformLike() {
	system := runtime.GOOS

	switch system {
	case "linux":
		platformInfo, err := host.Info()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get platform information")
			os.Exit(1)
		}
		switch platformInfo.Platform {
		case "ubuntu", "debian":
			PlatformLike = "debian"
		case "centos", "rhel", "redhat", "amazon", "fedora":
			PlatformLike = "rhel"
		default:
			log.Fatal().Msgf("Platform %s not supported", platformInfo.Platform)
		}
	case "windows", "darwin":
		PlatformLike = system
	default:
		log.Fatal().Msgf("Platform %s not supported", system)
	}
}

func JoinPath(base string, paths ...string) string {
	fullURL, err := url.JoinPath(base, paths...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to join path")
		return ""
	}

	return fullURL
}

func IsSuccessStatusCode(code int) bool {
	return code/100 == 2
}

func JoinUint64s(values []uint64) string {
	var strValues []string
	for _, value := range values {
		strValues = append(strValues, fmt.Sprintf("%d", value))
	}
	return strings.Join(strValues, ",")
}

// ScanBlock is a utility function that can be used to scan through text files
// that chunk using two-lined separators.
//
// Based on a function from the Datadog Agent.
// Original source : https://github.com/DataDog/datadog-agent
// License : Apache-2.0 license
func ScanBlock(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func GetEnvOrDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	return value
}

func ConvertGroupIds(groupIds []string) []uint32 {
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
