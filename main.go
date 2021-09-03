package main

import (
	"bufio"
	"log"
	"os/exec"
	"sync"
)

func main() {
	runCmd()
}

func runCmd() {
	var wg sync.WaitGroup
	bin := "/Users/al/tmp/kcptun/client_darwin_amd64"
	args := []string{"-c", "/Users/al/tmp/kcptun/la.json"}
	cmd := exec.Command(bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	outScanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)

	wg.Add(2)
	go func() {
		defer wg.Done()
		for outScanner.Scan() {
			text := outScanner.Text()
			log.Println(text)
		}
	}()

	go func() {
		defer wg.Done()
		for errScanner.Scan() {
			text := errScanner.Text()
			log.Println(text)
		}
	}()

	wg.Wait()

	defer killCmd(cmd)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func killCmd(cmd *exec.Cmd) {
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill: ", cmd.Process.Pid)
	}
}
