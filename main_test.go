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
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
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

// filepath.Walk reads the directory in another order then the scp command
// does - here is the order they need to be shipped in order to match the known test directory hash above
var TestDirectoryItems = []string{
	"test-directory/tekst.txt",
	"test-directory/levelone/doubleleveltwo/emptyfile",
	"test-directory/levelone/leveltwo/forest-sunbeams-trees-sunlight-70365.jpeg",
}

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
		for _, p := range TestDirectoryItems {
			// read test file
			fd, err := os.Open(p)
			if err != nil {
				t.Logf("could not read file: %s", err)
				t.Fail()
				break
			}

			// stat the file to figure out its size
			info, err := os.Stat(p)
			if err != nil {
				t.Logf("could not stat file %s: %s", p, err)
				t.Fail()
				break
			}

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s/%s/%s", viper.Get("ADVERTISE_URL"), "source", id, p), fd)
			req.ContentLength = info.Size()

			// http post test file
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				t.Logf("could not http post file: %s", err)
				t.Fail()
				break
			}

			// check status code
			if resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				t.Logf("wrong status code when uploading file (%s): %d: %s", p, resp.StatusCode, string(body))
				t.Fail()
			}
		}
		_, err = http.Get(fmt.Sprintf("%s%s/%s", viper.Get("ADVERTISE_URL"), "closesource", id))
		if err != nil {
			t.Logf("failed to close source: %s", err)
			t.Fail()
		}

		wg.Done()
	}()

	err = downloadKnownPayload(fmt.Sprintf("%s/sink/%s.tar.gz", viper.Get("ADVERTISE_URL"), id))
	if err != nil {
		t.Logf("known payload failed: %s", err)
		t.Fail()
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
		n, err := io.Copy(h, tarReader)
		if err != nil {
			return fmt.Errorf("could not update md5 digest: %s", err)
		}
		fmt.Printf("Added %s with %d bytes to md5sum\n", header.Name, n)
	}

	hexSum := fmt.Sprintf("%x", h.Sum(nil))
	if hexSum != KnownTestDirectoryHash {
		return fmt.Errorf("wrong md5 hash of test-directory zip file: %s != %s", hexSum, KnownTestDirectoryHash)
	}
	return nil
}
