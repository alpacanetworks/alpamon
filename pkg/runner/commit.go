package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	"io"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	commitURL = "/api/servers/servers/-/commit/"
	eventURL  = "/api/events/events/"

	passwdFilePath = "/etc/passwd"
	groupFilePath  = "/etc/group"

	dpkgFilePath = "/var/lib/dpkg/status"
	rpmFilePath  = "/var/lib/rpm/Packages"

	IFF_UP          = 1 << 0 // Interface is up
	IFF_LOOPBACK    = 1 << 3 // Loopback interface
	IFF_POINTOPOINT = 1 << 4 // Point-to-point link
	IFF_RUNNING     = 1 << 6 // Interface is running
)

var syncMutex sync.Mutex

func CommitAsync(session *scheduler.Session, commissioned bool) {
	if commissioned {
		go syncSystemInfo(session, nil)
	} else {
		go commitSystemInfo()
	}
}

func commitSystemInfo() {
	log.Info().Msg("Start committing system information.")

	data := collectData()

	scheduler.Rqueue.Put(commitURL, data, 80, time.Time{})
	scheduler.Rqueue.Post(eventURL, []byte(fmt.Sprintf(`{
		"reporter": "alpamon",
		"record": "committed", 
		"description": "Committed system information. version: %s"}`, version.Version)), 80, time.Time{})

	log.Info().Msg("Finished committing system information.")
}

func syncSystemInfo(session *scheduler.Session, keys []string) {
	syncMutex.Lock()
	defer syncMutex.Unlock()

	if len(keys) == 0 {
		for key := range commitDefs {
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		var currentData, remoteData any
		var err error

		entry, exists := commitDefs[key]
		if !exists {
			continue
		}

		switch key {
		case "server":
			loadAvg, err := getLoadAverage()
			if err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve load average")
			}
			currentData = &ServerData{
				Version: version.Version,
				Load:    loadAvg,
			}
			scheduler.Rqueue.Patch(utils.JoinPath(entry.URL, entry.URLSuffix), currentData, 80, time.Time{})
			continue
		case "info":
			if currentData, err = getSystemData(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve system info")
			}
			remoteData = &SystemData{}
		case "os":
			if currentData, err = getOsData(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve os info")
			}
			remoteData = &OSData{}
		case "time":
			if currentData, err = getTimeData(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve time info")
			}
			remoteData = &TimeData{}
		case "groups":
			if currentData, err = getGroupData(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve group info")
			}
			remoteData = &[]GroupData{}
		case "users":
			if currentData, err = getUserData(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve user info")
			}
			remoteData = &[]UserData{}
		case "interfaces":
			if currentData, err = getNetworkInterfaces(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve network interfaces")
			}
			remoteData = &[]Interface{}
		case "addresses":
			if currentData, err = getNetworkAddresses(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve network addresses")
			}
			remoteData = &[]Address{}
		case "packages":
			if currentData, err = getSystemPackages(); err != nil {
				log.Debug().Err(err).Msg("Failed to retrieve system packages")
			}
			remoteData = &[]SystemPackageData{}
		default:
			log.Debug().Msgf("Unknown key: %s", key)
			continue
		}

		resp, statusCode, err := session.Get(utils.JoinPath(entry.URL, entry.URLSuffix), 10)
		if statusCode == http.StatusOK {
			err = json.Unmarshal(resp, &remoteData)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal remote data")
				continue
			}
		} else if statusCode == http.StatusNotFound {
			remoteData = nil
		} else {
			log.Error().Err(err).Msgf("HTTP %d: Failed to get data for %s", statusCode, key)
			continue
		}

		if entry.MultiRow {
			dispatchComparison(entry, currentData, remoteData)
		} else {
			compareData(entry, currentData.(ComparableData), remoteData.(ComparableData))
		}
	}
}

func compareData(entry commitDef, currentData, remoteData ComparableData) {
	var createData, updateData interface{}

	if remoteData == nil {
		createData = currentData
	} else {
		if currentData != remoteData.GetData() {
			updateData = currentData
		}
	}
	if createData != nil {
		scheduler.Rqueue.Post(entry.URL, createData, 80, time.Time{})
	} else if updateData != nil {
		scheduler.Rqueue.Patch(entry.URL+remoteData.GetID()+"/", updateData, 80, time.Time{})
	}
}

func compareListData[T ComparableData](entry commitDef, currentData, remoteData []T) {
	currentMap := make(map[interface{}]ComparableData)
	for _, currentItem := range currentData {
		currentMap[currentItem.GetKey()] = currentItem
	}

	for _, remoteItem := range remoteData {
		if currentItem, exists := currentMap[remoteItem.GetKey()]; exists {
			if currentItem != remoteItem.GetData() {
				scheduler.Rqueue.Patch(entry.URL+remoteItem.GetID()+"/", currentItem.GetData(), 80, time.Time{})
			}
			delete(currentMap, currentItem.GetKey())
		} else {
			scheduler.Rqueue.Delete(entry.URL+remoteItem.GetID()+"/", nil, 80, time.Time{})
		}
	}

	var createData []interface{}
	for _, currentItem := range currentMap {
		createData = append(createData, currentItem.GetData())
	}
	if len(createData) > 0 {
		scheduler.Rqueue.Post(entry.URL, createData, 80, time.Time{})
	}
}

func collectData() *commitData {
	data := &commitData{}

	var err error
	data.Version = version.Version

	if data.Load, err = getLoadAverage(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve load average")
	}
	if data.Info, err = getSystemData(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve system info")
	}
	if data.OS, err = getOsData(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve os info")
	}
	if data.Time, err = getTimeData(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve time data")
	}
	if data.Users, err = getUserData(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve user data")
	}
	if data.Groups, err = getGroupData(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve group data")
	}
	if data.Interfaces, err = getNetworkInterfaces(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve network interfaces")
	}
	if data.Addresses, err = getNetworkAddresses(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve network addresses")
	}
	if data.Packages, err = getSystemPackages(); err != nil {
		log.Debug().Err(err).Msg("Failed to retrieve system packages")
	}

	return data
}

func getLoadAverage() (float64, error) {
	avg, err := load.Avg()
	if err != nil {
		return 0, err
	}
	return avg.Load1, nil
}

func getSystemData() (SystemData, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return SystemData{}, err
	}

	hostInfo, err := host.Info()
	if err != nil {
		return SystemData{}, err
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return SystemData{}, err
	}

	cpuPhysicalCores, err := cpu.Counts(false) // physical cores
	if err != nil {
		return SystemData{}, err
	}

	cpuLogicalCores, err := cpu.Counts(true) // logical cores
	if err != nil {
		return SystemData{}, err
	}

	return SystemData{
		UUID:             hostInfo.HostID,
		CPUType:          hostInfo.KernelArch,
		CPUBrand:         cpuInfo[0].ModelName,
		CPUPhysicalCores: cpuPhysicalCores,
		CPULogicalCores:  cpuLogicalCores,
		PhysicalMemory:   vm.Total,
		HardwareVendor:   cpuInfo[0].VendorID,
		HardwareModel:    cpuInfo[0].Model,
		HardwareSerial:   cpuInfo[0].PhysicalID,
		ComputerName:     hostInfo.Hostname,
		Hostname:         hostInfo.Hostname,
		LocalHostname:    hostInfo.Hostname,
	}, nil
}

func getOsData() (OSData, error) {
	major, minor, patch := 0, 0, 0

	hostInfo, err := host.Info()
	if err != nil {
		return OSData{}, err
	}

	versionParts := strings.Split(hostInfo.PlatformVersion, ".")
	if len(versionParts) > 0 {
		major, _ = strconv.Atoi(versionParts[0])
	}
	if len(versionParts) > 1 {
		minor, _ = strconv.Atoi(versionParts[1])
	}
	if len(versionParts) > 2 {
		patch, _ = strconv.Atoi(versionParts[2])
	}

	return OSData{
		Name:         hostInfo.Platform,
		Version:      hostInfo.PlatformVersion,
		Major:        major,
		Minor:        minor,
		Patch:        patch,
		Platform:     hostInfo.Platform,
		PlatformLike: utils.PlatformLike,
	}, nil
}

func getTimeData() (TimeData, error) {
	currentTime := time.Now()

	uptime, err := host.Uptime()
	if err != nil {
		return TimeData{}, err
	}

	timezone, _ := currentTime.Zone()

	return TimeData{
		Datetime: currentTime.Format(time.RFC3339),
		Timezone: timezone,
		Uptime:   uptime,
	}, nil
}

func getUserData() ([]UserData, error) {
	file, err := os.Open(passwdFilePath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to open passwd file")
		return nil, err
	}

	defer func() { _ = file.Close() }()

	users := []UserData{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			continue
		}

		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}

		users = append(users, UserData{
			Username:  fields[0],
			UID:       uid,
			GID:       gid,
			Directory: fields[5],
			Shell:     fields[6],
		})
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return users, nil
}

func getGroupData() ([]GroupData, error) {
	file, err := os.Open(groupFilePath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to open group file")
		return nil, err
	}
	defer func() { _ = file.Close() }()

	groups := []GroupData{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			continue
		}

		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		groups = append(groups, GroupData{
			GID:       gid,
			GroupName: fields[0],
		})
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func getNetworkInterfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces := []Interface{}
	for _, iface := range ifaces {
		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}
		interfaces = append(interfaces, Interface{
			Name:      iface.Name,
			Flags:     getFlags(iface),
			MTU:       iface.MTU,
			Mac:       mac,
			Type:      0, // TODO
			LinkSpeed: 0, // TODO
		})
	}

	return interfaces, nil
}

func getNetworkAddresses() ([]Address, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	addresses := []Address{}
	for _, iface := range ifaces {
		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			var mask net.IPMask
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				mask = v.Mask
			case *net.IPAddr:
				ip = v.IP
				mask = ip.DefaultMask()
			}
			if ip == nil || ip.To4() == nil {
				continue
			}
			addresses = append(addresses, Address{
				Address:       ip.To4().String(),
				Broadcast:     calculateBroadcastAddress(ip, mask),
				InterfaceName: iface.Name,
				Mask:          net.IP(mask).String(),
			})
		}
	}
	return addresses, nil
}

func getFlags(iface net.Interface) int {
	var flags int
	if iface.Flags&net.FlagUp != 0 {
		flags |= IFF_UP
	}
	if iface.Flags&net.FlagLoopback != 0 {
		flags |= IFF_LOOPBACK
	}
	if iface.Flags&net.FlagPointToPoint != 0 {
		flags |= IFF_POINTOPOINT
	}
	if iface.Flags&net.FlagRunning != 0 {
		flags |= IFF_RUNNING
	}
	return flags
}

func calculateBroadcastAddress(ip net.IP, mask net.IPMask) string {
	// only ipv4
	if ip.To4() == nil || len(mask) != net.IPv4len {
		return ""
	}

	broadcast := make(net.IP, len(ip.To4()))
	for i := 0; i < len(ip.To4()); i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast.String()
}

// Based on a function from the Datadog Agent.
// Original source : https://github.com/DataDog/datadog-agent
// License : Apache-2.0 license
func getSystemPackages() ([]SystemPackageData, error) {
	if utils.PlatformLike == "debian" {
		return getDpkgPackage()
	} else if utils.PlatformLike == "rhel" {
		return getRpmPackage()
	}

	return []SystemPackageData{}, nil
}

func getDpkgPackage() ([]SystemPackageData, error) {
	fd, err := os.Open(dpkgFilePath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to open dpkg file")
		return nil, err
	}
	defer func() { _ = fd.Close() }()

	var packages []SystemPackageData
	scanner := bufio.NewScanner(fd)
	scanner.Split(utils.ScanBlock)

	pkgNamePrefix := []byte("Package:")
	for scanner.Scan() {
		chunk := scanner.Bytes()
		lines := bytes.Split(chunk, []byte("\n"))

		var pkgName string
		for _, line := range lines {
			if bytes.HasPrefix(line, pkgNamePrefix) {
				pkgName = string(bytes.TrimSpace(line[len(pkgNamePrefix):]))
				break
			}
		}

		if pkgName == "" {
			continue
		}

		reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(chunk)))
		header, err := reader.ReadMIMEHeader()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		pkg := SystemPackageData{
			Name:    header.Get("Package"),
			Version: header.Get("Version"),
			Source:  header.Get("Source"),
			Arch:    header.Get("Architecture"),
		}

		packages = append(packages, pkg)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return packages, nil
}

func getRpmPackage() ([]SystemPackageData, error) {
	db, err := rpmdb.Open(rpmFilePath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to open rpm file")
		return nil, err
	}

	defer func() { _ = db.Close() }()

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}

	var packages []SystemPackageData
	for _, pkg := range pkgList {
		rpmPkg := SystemPackageData{
			Name:    pkg.Name,
			Version: pkg.Version,
			Source:  pkg.SourceRpm,
			Arch:    pkg.Arch,
		}

		packages = append(packages, rpmPkg)
	}

	return packages, nil
}

func dispatchComparison(entry commitDef, currentData, remoteData any) {
	switch v := remoteData.(type) {
	case *[]GroupData:
		compareListData(entry, currentData.([]GroupData), *v)
	case *[]UserData:
		compareListData(entry, currentData.([]UserData), *v)
	case *[]Interface:
		compareListData(entry, currentData.([]Interface), *v)
	case *[]Address:
		compareListData(entry, currentData.([]Address), *v)
	case *[]SystemPackageData:
		compareListData(entry, currentData.([]SystemPackageData), *v)
	}
}
