package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func TestMain(m *testing.M) {
	schttp, err := NewSchttp()
	if err != nil {
		panic(fmt.Sprintf("could not initialize schttp: %s", err))
	}

	schttp.scpServer = scp.NewServer()
	go schttp.scpServer.Listen(schttp.sshFd)

	schttp.webServer = web.NewServer()
	go schttp.webServer.Serve(schttp.httpFd)

	// run tests
	code := m.Run()

	schttp.scpServer.Shutdown("shutting down test server")
	schttp.webServer.Shutdown(context.TODO())

	os.Exit(code)
}

func TestHTTPSourceToZip(t *testing.T) {
	// first off we must acquire an ID to post files to
	u := fmt.Sprintf("%s%s", viper.Get("ADVERTISE_URL"), "newsource")
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("unable to get \"%s\": %s", u, err)
	}

	var idData map[string]string
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&idData)
	if err != nil {
		t.Fatalf("unable to parse id from newsource: %s", err)
	}
	resp.Body.Close()

	id := idData["ID"]
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := filepath.Walk("test-directory/", func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("unable to walk path %s: %s", p, info)
			}

			// skip directories
			if info.IsDir() {
				return nil
			}

			// read test file
			fd, err := os.Open(p)
			if err != nil {
				return fmt.Errorf("could not read file %s: %s", p, err)
			}

			// http post test file
			resp, err = http.Post(
				fmt.Sprintf("%s%s/%s/%s", viper.Get("ADVERTISE_URL"), "source", id, p),
				"application/binary",
				fd,
			)
			if err != nil {
				return fmt.Errorf("could not send file: %s", err)
			}

			// check status code
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("wrong status code when uploading file: %d", resp.StatusCode)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed walking test-directory: %s", err)
		}
		wg.Done()
	}()

	err = downloadKnownPayload(fmt.Sprintf("%s/sink/%s.tar.gz", viper.Get("ADVERTISE_URL"), id))
	if err != nil {
		t.Fatalf("known payload failed: %s", err)
	}
	wg.Wait()
}

// TestScpToZip tests schttp on a high level
// it tries to send files with standard unix tools
func TestScpToZip(t *testing.T) {
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

	err = downloadKnownPayload(url)
	if err != nil {
		t.Fatalf("known payload failed: %s", err)
	}
}

func downloadKnownPayload(url string) error {
	// download this url and compare its checksum against a known value
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to http get %s: %s", url, err)
	}

	defer response.Body.Close()

	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		return fmt.Errorf("unable to read gzip: %s", err)
	}

	tarReader := tar.NewReader(gzipReader)

	h := md5.New()

	for {

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar error: %s", err)
		}

		// first just append the name of the file
		_, err = h.Write([]byte(header.Name))
		if err != nil {
			return fmt.Errorf("could not update md5 digest: %s", err)
		}

		// then the whole content of the file
		_, err = io.Copy(h, tarReader)
		if err != nil {
			return fmt.Errorf("could not update md5 digest: %s", err)
		}
	}

	hexSum := fmt.Sprintf("%x", h.Sum(nil))
	if hexSum != KnownTestDirectoryHash {
		return fmt.Errorf("wrong md5 hash of test-directory zip file: %s != %s", hexSum, KnownTestDirectoryHash)
	}
	return nil
}
