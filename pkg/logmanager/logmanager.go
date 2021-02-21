package logmanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
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

	var wg sync.WaitGroup
	ch := make(chan string, len(servers))

	log.Print(servers)
	for _, srv := range servers {
		wg.Add(1)
		go loadLog(srv, ch, &wg)
	}
	wg.Wait()
	close(ch)

	f, err := os.Create("/app/jet/scripts/klassen/psaccesslog.txt")
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

func loadLog(srv server, ch chan string, wg *sync.WaitGroup) error {
	defer wg.Done()
	log.Print("loadLog started")
	client, err := getSSHClient(srv)
	if err != nil {
		return err
	}
	log.Printf("ssh client for %s created", srv.Name)

	const timeLayout = "02\\/Jan\\/2006:15:04:05"
	tenMinutesAgo := time.Now().Add(time.Minute * -10)

	cuttedLog, err := sshDo(client,
		fmt.Sprintf(`tail -c 100000000 /app/nginx/logs/atg-access.log | sed -n "/%s/,$ p"`, tenMinutesAgo.Format(timeLayout)))
	if err != nil {
		return err
	}
	log.Printf("raw log from %s taken", srv.Name)

	ch <- string(cuttedLog)
	log.Printf("good log from %s sent to chan", srv.Name)

	return nil
}

func sshDo(sshClient *ssh.Client, cmd string) ([]byte, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	log.Printf("session for cmd '%s' created", cmd)

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, err
	}
	log.Printf("len of session output: %d", len(output))

	return output, nil
}

func getSSHClient(srv server) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: srv.Host.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(srv.Host.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := srv.Host.Addr + ":" + srv.Host.Port
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}
	log.Printf("connected to server addr: %s", addr)

	return sshClient, nil
}
