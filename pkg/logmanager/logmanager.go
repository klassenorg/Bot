package logmanager

import (
	"bufio"
	"encoding/json"
	"log"
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
	log.Print("config parsed: ", cfg)

	srvList := make([]server, len(servers))
	for i, srv := range servers {
		srvList[i] = cfg.Servers[srv]
	}
	log.Print("servers list parsed: ", srvList)

	err = writeLogToFile(srvList)
	if err != nil {
		return err
	}
	log.Print("log writted to file")

	return nil
}

func parseConfig(configPath string) (*config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return &config{}, err
	}
	log.Printf("file %s opened", configPath)
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
	log.Printf("file created")

	w := bufio.NewWriter(f)
	for log := range ch {
		_, err := w.WriteString(log)
		if err != nil {
			return err
		}
	}
	w.Flush()
	log.Print("log writed in file")

	return nil
}

func loadLog(srv server, ch chan string) error {
	client, err := getSSHClient(srv)
	if err != nil {
		return err
	}
	log.Printf("ssh client for %s created", srv.Name)

	rawLog, err := sshDo(client,
		"tail -c 100000000 /app/nginx/logs/atg-access.log")
	if err != nil {
		return err
	}
	log.Printf("raw log from %s taken", srv.Name)

	goodLog := cutLog(rawLog)

	ch <- goodLog
	log.Printf("good log from %s sent to chan", srv.Name)

	return nil
}

func cutLog(rawLog string) string {
	tenMinutesAgo := time.Now().Add(time.Minute * -10)

	const timeLayout = "[02/Jan/2006:15:04:05"

	lines := strings.Split(rawLog, "\n")
	log.Print("raw log splitted")

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
	log.Printf("len of goodLines: %d", len(goodLines))

	return goodLines
}

func sshDo(sshClient *ssh.Client, cmd string) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", err
	}
	log.Printf("session for cmd '%s' created", cmd)

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	log.Printf("len of session output: %d", len(output))

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
	log.Printf("connected to server addr: %s", addr)

	return sshClient, nil
}
