package utils

import (
	"regexp"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
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

func IsVirtualFileSystem(mountPoint string) bool {
	pattern := "^/(sys|proc|run|dev/)"
	matched, _ := regexp.MatchString(pattern, mountPoint)
	if matched {
		return true
	}

	virtualMountpoints := map[string]bool{
		"/sys":  true,
		"/proc": true,
		"/dev":  true,
	}

	if virtualMountpoints[mountPoint] {
		return true
	}

	return false
}
