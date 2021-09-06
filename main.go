package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	latestReleaseUrl = "https://api.github.com/repos/xtaci/kcptun/releases/latest"
	WinPkg           = "-windows-amd64-"
	LinuxPkg         = "-linux-amd64-"
	MacPkg           = "-darwin-amd64-"
)

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	binPath, err := download(path)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(binPath)
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
		if strings.Contains(asset.Name, getTargetPkgName()) {
			obj = asset
			break
		}
	}

	r, err := http.Get(obj.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	if r.StatusCode != 200 {
		return "", errors.New(r.Status)
	}
	defer r.Body.Close()

	workDir := filepath.Join(dir, "bin")
	_ = os.MkdirAll(workDir, os.ModePerm)
	p := filepath.Join(dir, "bin", obj.Name)
	out, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err = io.Copy(out, r.Body); err != nil {
		return "", err
	}

	file, err := os.Open(p)
	if err != nil {
		return "", err
	}

	p, err = ExtractTarGz(file, workDir)
	if err != nil {
		return "", err
	}

	return p, nil
}

func ExtractTarGz(gzipStream io.Reader, parentFolder string) (string, error) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	bin := ""
	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		p := filepath.Join(parentFolder, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(p, 0755); err != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, err := os.Create(p)
			if err != nil {
				log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
			}
			if strings.Contains(p, "client_") {
				bin = p
			}
			outFile.Close()

		default:
			log.Fatalf("ExtractTarGz: uknown type: %s in %s", header.Typeflag, header.Name)
		}

	}

	return bin, nil
}

func getTargetPkgName() string {
	switch runtime.GOOS {
	case "windows":
		return WinPkg
	case "linux":
		return LinuxPkg
	case "darwin":
		return MacPkg
	default:
		log.Fatalf("No platform found.")
	}
	return ""
}
