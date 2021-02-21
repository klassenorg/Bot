package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/klassenorg/Bot/pkg/logmanager"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("Wrong arguments"))
	}
	args := os.Args[1]

	parts := strings.Split(args, ",")

	servers := make([]int, 0)

	for _, part := range parts {
		if strings.Contains(part, "-") {
			server := strings.Split(part, "-")
			left, _ := strconv.Atoi(server[0])
			right, _ := strconv.Atoi(server[1])
			for i := left; i <= right; i++ {
				if !contains(servers, i) && i < 23 && i > 0 {
					servers = append(servers, i-1)
				}
			}
		} else {
			i, _ := strconv.Atoi(part)
			if !contains(servers, i) && i < 23 && i > 0 {
				servers = append(servers, i-1)
			}
		}
	}
	err := logmanager.Run("/home/klassen/Bot/config.json", servers)
	if err != nil {
		log.Fatal(err)
	}
}

func contains(i []int, j int) bool {
	for _, v := range i {
		if v == j {
			return true
		}
	}
	return false
}
