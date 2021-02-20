package logmanager

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type config struct {
	Servers []server
}

type server struct {
	Name string
	Host host
}

type host struct {
	Addr     string
	User     string
	Password string
	Port     string
}

//Run parses servers config from file and insert 10 minutes from choosen servers to file.
func Run(configPath string, servers []int) error {

	cfg, err := parseConfig(configPath)
	if err != nil {
		return err
	}

	srvList := make([]server, len(servers))
	for i, srv := range servers {
		srvList[i] = cfg.Servers[srv]
	}

	err = writeLogToFile(srvList)
	if err != nil {
		return err
	}

	return nil
}

func parseConfig(configPath string) (*config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return &config{}, err
	}
	decoder := json.NewDecoder(file)
	cfg := new(config)
	err = decoder.Decode(&cfg)
	if err != nil {
		return &config{}, err
	}

	return cfg, nil
}

func writeLogToFile(servers []server) error {

	ch := make(chan string, 22)

	for _, srv := range servers {
		go loadLog(srv, ch)
	}

	f, err := os.Create("/app/jet/scripts/klassen/psacceslogGo.txt")
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	for log := range ch {
		_, err := w.WriteString(log)
		if err != nil {
			return err
		}
	}
	w.Flush()

	return nil
}

func loadLog(srv server, ch chan string) error {
	client, err := getSSHClient(srv)
	if err != nil {
		return err
	}

	rawLog, err := sshDo(client,
		"tail -c 100000000 /app/nginx/logs/atg-access.log")
	if err != nil {
		return err
	}

	goodLog := cutLog(rawLog)

	ch <- goodLog

	return nil
}

func cutLog(rawLog string) string {
	tenMinutesAgo := time.Now().Add(time.Minute * -10)

	const timeLayout = "[02/Jan/2006:15:04:05"

	lines := strings.Split(rawLog, "\n")

	var goodLines string

	for lineNum, line := range lines {
		currentLine := lines[len(lines)-1-lineNum]
		if len(currentLine) < 50 {
			continue
		}
		timeStamp, err := time.Parse(timeLayout, strings.Split(currentLine, " ")[2])
		if err != nil {
			continue
		}
		if timeStamp.Before(tenMinutesAgo) {
			break
		}
		goodLines = line + goodLines
	}

	return goodLines
}

func sshDo(sshClient *ssh.Client, cmd string) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", err
	}

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func getSSHClient(srv server) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: srv.Host.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(srv.Host.Password),
		},
	}

	addr := srv.Host.Addr + ":" + srv.Host.Port
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return sshClient, nil
}
