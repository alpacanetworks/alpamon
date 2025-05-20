package utils

import (
	"regexp"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
)

var (
	virtualFileSystems = map[string]bool{
		"tmpfs":       true,
		"devtmpfs":    true,
		"proc":        true,
		"sysfs":       true,
		"cgroup":      true,
		"cgroup2":     true,
		"overlay":     true,
		"autofs":      true,
		"devfs":       true,
		"securityfs":  true,
		"fusectl":     true,
		"hugetlbfs":   true,
		"debugfs":     true,
		"pstore":      true,
		"tracefs":     true,
		"devpts":      true,
		"mqueue":      true,
		"bpf":         true,
		"configfs":    true,
		"binfmt_misc": true,
	}
	virtualMountpoints = map[string]bool{
		"/sys":  true,
		"/proc": true,
		"/dev":  true,
	}
	virtualMountPointPattern = "^/(sys|proc|run|dev/)"
	virtaulDisk              = map[string]bool{
		"loop": true,
		"ram":  true,
		"fd":   true,
		"sr":   true,
		"zram": true,
	}
	loopFileSystemPrefix = "/dev/loop"
	linuxDiskNamePattern = regexp.MustCompile(`^([a-z]+[0-9]*)(p[0-9]+)?$`)
	macDiskNamePattern   = regexp.MustCompile(`^(disk[0-9]+)(s[0-9]+)?$`)
)

func CalculateNetworkBps(current net.IOCountersStat, last net.IOCountersStat, interval time.Duration) (inputBps float64, outputBps float64) {
	if interval == 0 {
		return 0, 0
	}

	inputBytesDiff := float64(current.BytesRecv - last.BytesRecv)
	outputBytesDiff := float64(current.BytesSent - last.BytesSent)
	seconds := interval.Seconds()

	inputBps = (inputBytesDiff * 8) / seconds
	outputBps = (outputBytesDiff * 8) / seconds

	return inputBps, outputBps
}

func CalculateNetworkPps(current net.IOCountersStat, last net.IOCountersStat, interval time.Duration) (inputPps float64, outputPps float64) {
	if interval == 0 {
		return 0, 0
	}

	inputPktsDiff := float64(current.PacketsRecv - last.PacketsRecv)
	outputPktsDiff := float64(current.PacketsSent - last.PacketsSent)
	seconds := interval.Seconds()

	inputPps = inputPktsDiff / seconds
	outputPps = outputPktsDiff / seconds

	return inputPps, outputPps
}

func CalculateDiskIOBps(current disk.IOCountersStat, last disk.IOCountersStat, interval time.Duration) (readBps float64, writeBps float64) {
	if interval == 0 {
		return 0, 0
	}

	readBytesDiff := float64(current.ReadBytes - last.ReadBytes)
	writeBytesDiff := float64(current.WriteBytes - last.WriteBytes)
	seconds := interval.Seconds()

	readBps = readBytesDiff / seconds
	writeBps = writeBytesDiff / seconds

	return readBps, writeBps
}

func IsVirtualFileSystem(device string, fstype string, mountPoint string) bool {
	if strings.HasPrefix(device, loopFileSystemPrefix) {
		return true
	}

	matched, _ := regexp.MatchString(virtualMountPointPattern, mountPoint)
	if matched {
		return true
	}

	if virtualFileSystems[fstype] {
		return true
	}

	if virtualMountpoints[mountPoint] {
		return true
	}

	return false
}

func IsVirtualDisk(name string) bool {
	if virtaulDisk[name] {
		return true
	}

	return false
}

func ParseDiskName(device string) string {
	device = strings.TrimPrefix(device, "/dev/")

	re := regexp.MustCompile(`^[a-zA-Z]+\d*`)
	if match := re.FindString(device); match != "" {
		return match
	}

	for i := len(device) - 1; i >= 0; i-- {
		if device[i] < '0' || device[i] > '9' {
			return device[:i+1]
		}
	}

	return device
}

func GetDiskBaseName(name string) string {
	if matches := linuxDiskNamePattern.FindStringSubmatch(name); len(matches) == 2 {
		return matches[1]
	}

	if matches := macDiskNamePattern.FindStringSubmatch(name); len(matches) == 2 {
		return matches[1]
	}

	return name
}
