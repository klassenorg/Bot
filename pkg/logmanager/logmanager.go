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
	log.Printf("Config parsed successfully, found %d servers", len(cfg.Servers))
	for _, srvForLog := range cfg.Servers {
		log.Printf("%s %s", srvForLog.Name, srvForLog.Host.Addr)
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

	w := bufio.NewWriter(f)
	for log := range ch {
		_, err := w.WriteString(log)
		if err != nil {
			return err
		}
	}
	w.Flush()
	log.Println("file saved")

	return nil
}

func loadLog(srv server, ch chan string, wg *sync.WaitGroup) error {
	defer wg.Done()
	client, err := getSSHClient(srv)
	if err != nil {
		return err
	}
	defer client.Close()

	const timeLayout = "02/Jan/2006:15:04:"
	tenMinutesAgo := time.Now().Add(time.Minute * -11)
	cmd := fmt.Sprintf(`tail -n $(tac /app/nginx/logs/atg-access.log | grep -Fnm 1 '%s' | awk -F ":" '{printf $1;}') /app/nginx/logs/atg-access.log`, tenMinutesAgo.Format(timeLayout))

	log.Printf("command %s sent to %s", cmd, srv.Name)
	cuttedLog, err := sshDo(client, cmd)
	if err != nil {
		return err
	}

	ch <- srv.Name + "\n" + string(cuttedLog)
	log.Printf("output sent from %s to channel", srv.Name)

	return nil
}

func sshDo(sshClient *ssh.Client, cmd string) ([]byte, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, err
	}

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

	return sshClient, nil
}
