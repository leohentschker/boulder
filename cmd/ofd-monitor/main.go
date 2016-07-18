package main

import (
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/metrics"
)

type config struct {
	Programs map[string]string // program name: pid file path
}

func readPF(path string) (int, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpaces(string(bytes)))
}

func checkOFD(pid int) (int, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return 0, err
	}
	names, err := f.Readdirnames(0)
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func main() {
	cPath := flag.String("config", "", "")
	d := flag.Duration("interval", time.Minute, "")
	statsdAddr := flag.String("statsdAddr", "localhost:8125", "")
	flag.Parse()

	cBytes, err := ioutil.ReadFile(cPath)
	if err != nil {
		panic(err)
	}
	var c config
	err := json.Unmarshal(cBytes, &c)
	if err != nil {
		panic(err)
	}

	m, err := metrics.NewStatter(*statsdAddr, "Boulder")
	if err != nil {
		panic(err)
	}

	for n, p := range c.Programs {
		go func(name, path string) {
			for {
				pid, err := readPF(path)
				if err != nil {
					// log
					continue
				}
				ofd, err := checkOFD(pid)
				if err != nil {
					// log
					continue
				}
				m.Gauge(fmt.Sprintf("OpenFD.%s.%d", name, pid), ofd, 1.0)
				time.Sleep(*d)
			}
		}(n, p)
	}
}