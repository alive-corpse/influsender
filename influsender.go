package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	s "strings"
	"syscall"

	"gopkg.in/ini.v1"
)

var (
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func Init(debugHandle, infoHandle, warningHandle, errorHandle io.Writer) { // Removed |log.Lshortfile
	Debug = log.New(debugHandle, "[D] ", log.Ldate|log.Ltime)
	Info = log.New(infoHandle, "[I] ", log.Ldate|log.Ltime)
	Warning = log.New(warningHandle, "[W] ", log.Ldate|log.Ltime)
	Error = log.New(errorHandle, "[E] ", log.Ldate|log.Ltime)
}

// func ld(msg string) { Debug.Println(msg) }
func li(msg string) { Info.Println(msg) }
func lw(msg string) { Warning.Println(msg) }
func le(msg string) { Error.Println(msg) }

var hostname string

// Read from file
func r(fname string) string {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		le("Fail to read from file " + fname)
		panic(err)
	}
	return s.Trim(string(content), " \n")
}

// Formatting content
func f(name, value, tags string) string {
	if name == "" || value == "" {
		lw("Can't format content without name of value")
		return ""
	} else {
		if len(tags) != 0 {
			tags = "," + tags
		}
		return fmt.Sprintf("%s,host=%s%s value=%s", name, hostname, tags, value)
	}
}

// Getting load average
func loadavg() string {
	res := ""
	li("Getting load average...")
	loadavg := s.Split(r("/proc/loadavg"), " ")[0:3]
	res += f("LoadAvg", loadavg[0], "period=1m") + "\n"
	res += f("LoadAvg", loadavg[1], "period=10m") + "\n"
	res += f("LoadAvg", loadavg[2], "period=15m") + "\n"
	return res
}

func uptime() string {
	li("Getting uptime...")
	return f("UpTime", s.Fields(r("/proc/uptime"))[0], "") + "\n"
}

// Getting memory stats
func mem() string {
	res := ""
	li("Getting memory stats")
	lines := s.Split(r("/proc/meminfo"), "\n")
	for i := 0; i < len(lines); i++ {
		fields := s.Split(lines[i], ":")
		switch fields[0] {
		case "MemTotal", "MemFree", "MemAvailable", "Buffers", "Cached",
			"Shmem", "KernelStack", "VmallocTotal", "VmallocUsed", "VmallocChunk":
			val, err := strconv.ParseFloat(s.Split(s.Trim(fields[1], " "), " ")[0], 64)
			if err != nil {
				lw("Can't format memory to int for " + fields[0])
			} else {
				res += f("Mem", fmt.Sprintf("%.2f", val/1024), "memtype="+fields[0]) + "\n"
			}
		}
	}
	return res
}

func df(path string) string {
	res := ""
	if path == "" {
		path = "/"
	}
	p := ""
	paths := s.Split(path, ",")
	for i := 0; i < len(paths); i++ {
		p = s.Trim(paths[i], " ")
		li("Getting disk status for path " + p)
		stat := syscall.Statfs_t{}
		err := syscall.Statfs(p, &stat)
		if err != nil {
			lw("Fail to make syscall for getting file system statistics for " + path)
			return ""
		}
		var dfree, davail, dtotal, dused, dfreeprec, davailprec, dusedprec float64
		dfree = float64(stat.Bfree) * float64(stat.Bsize) / 1024 / 1024 / 1024
		davail = float64(stat.Bavail) * float64(stat.Bsize) / 1024 / 1024 / 1024
		dtotal = float64(stat.Blocks) * float64(stat.Bsize) / 1024 / 1024 / 1024
		dused = dtotal - dfree
		dfreeprec = dfree / dtotal * 100
		davailprec = davail / dtotal * 100
		dusedprec = 100 - davailprec
		res += f("Disk", fmt.Sprintf("%.3f", dfree), "disktype=free,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.2f", dfreeprec), "disktype=freeprec,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.3f", davail), "disktype=avail,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.2f", davailprec), "disktype=availprec,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.3f", dused), "disktype=used,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.2f", dusedprec), "disktype=usedprec,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%d", stat.Ffree), "disktype=ifree,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%d", stat.Files-stat.Ffree), "disktype=iused,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.2f", float64(stat.Ffree)/float64(stat.Files)*100), "disktype=ifreeprec,path="+p) + "\n"
		res += f("Disk", fmt.Sprintf("%.2f", float64((stat.Files-stat.Ffree))/float64(stat.Files)*100), "disktype=iusedprec,path="+p) + "\n"
	}

	return res
}

func dio(disks string) string {
	res := ""

	var diskNameCheck = regexp.MustCompile(`^([sh]d[a-z]+|mmcblk[0-9]+)$`)
	wl := s.Split(disks, ",") // White list of disks
	d := s.Split(r("/proc/diskstats"), "\n")
	for i := 0; i < len(d); i++ {
		dfields := s.Fields(d[i])
		// lc := dfields[2][len(dfields[2])-1]
		// if lc < uint8(48) || lc > uint8(57) {
		if diskNameCheck.MatchString(dfields[2]) {
			li("Getting disk io stats for " + dfields[2])
			if len(wl[0]) > 0 {
				for wli := 0; wli < len(wl); wli++ {
					if wl[wli] == dfields[2] {
						// bs:= strconv.ParseUint(r("/sys/block/" + dfields[2] + "/queue/physical_block_size"),10,64) // Sector size
						res += f("Disk", dfields[3], "disktype=reads,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[4], "disktype=mergedreads,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[5], "disktype=sectorsreads,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[6], "disktype=readingms,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[7], "disktype=writes,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[8], "disktype=mergedwrites,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[9], "disktype=sectorswrites,dev="+dfields[2]) + "\n"
						res += f("Disk", dfields[10], "disktype=writingms,dev="+dfields[2]) + "\n"
					}
				}
			} else {
				res += f("Disk", dfields[3], "disktype=reads,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[4], "disktype=mergedreads,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[5], "disktype=sectorsreads,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[6], "disktype=readingms,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[7], "disktype=writes,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[8], "disktype=mergedwrites,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[9], "disktype=sectorswrites,dev="+dfields[2]) + "\n"
				res += f("Disk", dfields[10], "disktype=writingms,dev="+dfields[2]) + "\n"
			}
		}
	}
	return res
}

func net(ifaceswhitelist string) string {
	res := ""
	li("Getting network interface(s) stats " + ifaceswhitelist)
	interfaces := s.Split(r("/proc/net/dev"), "\n")[2:]
	wl := s.Split(ifaceswhitelist, ",")
	for i := 0; i < len(interfaces); i++ {
		iface := s.Fields(interfaces[i])
		iname := s.Replace(iface[0], ":", "", 1)
		if len(wl[0]) > 0 {
			for wli := 0; wli < len(wl); wli++ {
				if wl[wli] == iname {
					res += f("Network", iface[1], "nettype=ibytes,iface="+iname) + "\n"
					res += f("Network", iface[2], "nettype=ipackets,iface="+iname) + "\n"
					res += f("Network", iface[3], "nettype=ierrors,iface="+iname) + "\n"
					res += f("Network", iface[4], "nettype=idrop,iface="+iname) + "\n"
					res += f("Network", iface[9], "nettype=obytes,iface="+iname) + "\n"
					res += f("Network", iface[10], "nettype=opackets,iface="+iname) + "\n"
					res += f("Network", iface[11], "nettype=oerrors,iface="+iname) + "\n"
					res += f("Network", iface[12], "nettype=odrop,iface="+iname) + "\n"
				}
			}
		} else {
			res += f("Network", iface[1], "nettype=ibytes,iface="+iname) + "\n"
			res += f("Network", iface[2], "nettype=ipackets,iface="+iname) + "\n"
			res += f("Network", iface[3], "nettype=ierrors,iface="+iname) + "\n"
			res += f("Network", iface[4], "nettype=idrop,iface="+iname) + "\n"
			res += f("Network", iface[9], "nettype=obytes,iface="+iname) + "\n"
			res += f("Network", iface[10], "nettype=opackets,iface="+iname) + "\n"
			res += f("Network", iface[11], "nettype=oerrors,iface="+iname) + "\n"
			res += f("Network", iface[12], "nettype=odrop,iface="+iname) + "\n"
		}

	}
	return res
}

func temp() string {
	res := ""
	_, err := os.Stat("/sys/class/thermal/")
	if err == nil {
		li("Getting temperature")
		dir, err := os.Open("/sys/class/thermal/")
		if err != nil {
			lw("Can't read /sys/class/thermal directory")
			return ""
		}
		dirs, err := dir.Readdir(-1)
		if err != nil {
			lw("Something goes wrong while reading directory /sys/calss/thermal")
			return ""
		}
		dir.Close()
		thermalZone := regexp.MustCompile(`^thermal_zone[0-9]+$`)
		for _, d := range dirs {
			if thermalZone.MatchString(d.Name()) {
				zone := d.Name()[12:]
				t := r("/sys/class/thermal/thermal_zone" + zone + "/temp")
				ttype := s.ReplaceAll(s.Trim(r("/sys/class/thermal/thermal_zone"+zone+"/type"), " "), " ", "_")
				if ttype == "" {
					ttype = "none"
				}
				temp := ""
				if len(t) > 1 {
					temp = s.TrimRight(t[0:len(t)-3]+"."+t[len(t)-3:], "0")
					if s.HasSuffix(temp, ".") {
						temp = temp + "0"
					}
				} else {
					temp = t
				}
				res += f("Temp", temp, "temptype="+ttype+",zone="+zone) + "\n"
			}
		}
		return res
	}
	return ""
}

func proc() string {
	li("Counting processes...")
	isIntCheck := regexp.MustCompile(`^[0-9]+$`)
	dir, err := os.Open("/proc/")
	if err != nil {
		lw("Can't read /proc directory")
		return ""
	}
	files, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		lw("Can't read list of processes")
		return ""
	}
	count := 0
	for _, file := range files {
		if isIntCheck.MatchString(file.Name()) {
			count++
		}
	}
	return f("Processes", strconv.Itoa(count), "") + "\n"
}

func send(url, data string) string {
	if url == "" {
		le("Empty influxdb url")
		return ""
	}
	if url == "" {
		le("Nothing to send ot influxdb")
		return ""
	}
	li("Sending data to remote server")
	client := &http.Client{}
	payload := s.NewReader(data)
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Add("Content-Type", "text/plain")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(body)
}

func main() {
	Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	hostname = r("/etc/hostname")

	li("Start influsender process on host " + hostname)

	proto := ""
	host := ""
	port := ""
	database := ""
	checklist := ""
	mountpoints := ""
	disks := ""
	interfaces := ""

	_, err := os.Stat("influsender.ini")
	if err == nil {
		li("Config file influsender.ini found")
		cfg, err := ini.Load("influsender.ini")
		if err != nil {
			fmt.Printf("Fail to read file: %v", err)
		} else {
			// Default parameters
			proto = cfg.Section("influxdb").Key("protocol").String()
			host = cfg.Section("influxdb").Key("host").String()
			port = cfg.Section("influxdb").Key("port").String()
			database = cfg.Section("influxdb").Key("database").String()
			checklist = cfg.Section("main").Key("checklist").String()
			mountpoints = cfg.Section("disk").Key("mountpoints").String()
			disks = cfg.Section("diskio").Key("disks").String()
			interfaces = cfg.Section("net").Key("interfaces").String()
		}
	}

	if proto == "" {
		proto = "http"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8086"
	}
	if database == "" {
		database = "influsender"
	}
	if checklist == "" {
		checklist = "uptime,loadavg,mem,disk,diskio,net,temp,proc"
	}
	url := fmt.Sprintf("%s://%s:%s/write?db=%s", proto, host, port, database)

	res := ""
	item := ""
	cl := s.Split(checklist, ",")
	for i := 0; i < len(cl); i++ {
		item = s.Trim(cl[i], " ")
		switch item {
		case "uptime":
			res += uptime()
		case "loadavg":
			res += loadavg()
		case "mem":
			res += mem()
		case "disk":
			res += df(mountpoints)
		case "diskio":
			res += dio(disks)
		case "net":
			res += net(interfaces)
		case "temp":
			res += temp()
		case "proc":
			res += proc()
		}
	}

	send(url, res)
}
