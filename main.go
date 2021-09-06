package main

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const latestReleaseUrl = "https://api.github.com/repos/xtaci/kcptun/releases/latest"

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	binPath, err := download(path)
	if err != nil {
		log.Fatal(err)
	}
	return
	runCmd(binPath)
}

func runCmd(bin string) {
	var wg sync.WaitGroup
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

func download(dir string) (string, error) {
	resp, err := http.Get(latestReleaseUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := IReleaseResp{}
	_ = json.Unmarshal(body, &result)
	var obj IReleaseAsset
	for _, asset := range result.Assets {
		if strings.Contains(asset.Name, "-darwin-amd64-") {
			obj = asset
			break
		}
	}

	r, err := http.Get(obj.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	p := dir + "/" + obj.Name
	out, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err = io.Copy(out, r.Body); err != nil {
		return "", err
	}

	return p, nil
}
