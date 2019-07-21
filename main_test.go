package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fasmide/schttp/scp"
	"github.com/fasmide/schttp/web"
	"github.com/spf13/viper"
)

// KnownTestDirectoryHash is the md5 digest of test-directory filenames and contents
const KnownTestDirectoryHash = "3e8a7eddba29589d389dc65082f840c6"

var scpPort, httpPort int

// init sets up some environment variables for testing
func init() {

	// we dont need secure randomness
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	// we use random ports in the range 30000:uint16
	scpPort = r.Intn(35535) + 30000
	httpPort = r.Intn(35535) + 30000

	viper.SetDefault("HTTP_LISTEN", fmt.Sprintf("127.0.0.1:%d", httpPort))
	viper.SetDefault("ADVERTISE_URL", fmt.Sprintf("http://127.0.0.1:%d/", httpPort))
	viper.SetDefault("SSH_LISTEN", fmt.Sprintf("127.0.0.1:%d", scpPort))
	viper.SetDefault("PID_FILE", "/tmp/test_schttp.pid")
}

// TestMain tests schttp on a high level
// it tries to send files with standard unix tools
func TestMain(t *testing.T) {
	schttp, err := NewSchttp()
	if err != nil {
		t.Fatalf("could not initialize schttp: %s", err)
	}

	var shutdown sync.WaitGroup

	shutdown.Add(1)
	go func() {
		schttp.scpServer = scp.NewServer()
		go schttp.scpServer.Listen(schttp.sshFd)

		schttp.webServer = &web.Server{}
		go schttp.webServer.Listen(schttp.httpFd)

		// wait here till we are finished testing
		shutdown.Wait()

		schttp.scpServer.Shutdown("shutting down test server")
		schttp.webServer.Shutdown(context.TODO())
	}()

	scp := exec.Command("scp",
		"-oStrictHostKeyChecking=no",
		fmt.Sprintf("-P%d", scpPort),
		"-r", "test-directory/", "127.0.0.1:",
	)

	reader, err := scp.StderrPipe()
	if err != nil {
		t.Fatalf("could not get stderr pipe from scp command: %s", err)
	}

	defer reader.Close()

	t.Log("Executing ", scp.Args)
	scp.Start()

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var url string
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), "\n ")
		if strings.HasSuffix(line, ".tar.gz") {
			// we found our string
			url = line
			break
		}
	}

	if url == "" {
		t.Fatalf("no url found in stderr from scp")
	}

	// download this url and compare its checksum against a known value
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("unable to http get %s: %s", url, err)
	}

	defer response.Body.Close()

	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("unable to read gzip: %s", err)
	}

	tarReader := tar.NewReader(gzipReader)

	h := md5.New()

	for {

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar error: %s", err)
		}

		// first just append the name of the file
		_, err = h.Write([]byte(header.Name))
		if err != nil {
			t.Fatalf("could not update md5 digest: %s", err)
		}

		// then the whole content of the file
		_, err = io.Copy(h, tarReader)
		if err != nil {
			t.Fatalf("could not update md5 digest: %s", err)
		}
	}

	hexSum := fmt.Sprintf("%x", h.Sum(nil))
	if hexSum != KnownTestDirectoryHash {
		t.Fatalf("wrong md5 hash of test-directory zip file: %s != %s", hexSum, KnownTestDirectoryHash)
	}

	shutdown.Done()
}
