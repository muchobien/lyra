package e2e

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

const garageImage = "dxflrs/garage:v2.5.0@sha256:677e5b758e4425a325b811c425c026bf35a4bf42f7ddbaa773de3f4fd779a5a5"

func freePort(test *testing.T) string {
	test.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		test.Fatal(err)
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	return port
}

func TestS3Storage(test *testing.T) {
	if err := exec.Command("docker", "info").Run(); err != nil {
		test.Skip("needs Docker to run Garage")
	}
	dir := test.TempDir()
	secret := make([]byte, 32)
	rand.Read(secret)
	config := fmt.Sprintf(`metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "[::]:3901"
rpc_secret = "%x"
[s3_api]
s3_region = "garage"
api_bind_addr = "[::]:3900"
`, secret)
	if err := os.WriteFile(filepath.Join(dir, "garage.toml"), []byte(config), 0o644); err != nil {
		test.Fatal(err)
	}
	s3Port := freePort(test)
	container := "lyra-e2e-garage-" + s3Port
	run := func(args ...string) string {
		test.Helper()
		output, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			test.Fatalf("docker %s: %v: %s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}
	garage := func(args ...string) string {
		test.Helper()
		return run(append([]string{"exec", container, "/garage"}, args...)...)
	}
	run("run", "-d", "--name", container, "-p", "127.0.0.1:"+s3Port+":3900", "-v", filepath.Join(dir, "garage.toml")+":/etc/garage.toml", garageImage)
	test.Cleanup(func() { exec.Command("docker", "rm", "-f", container).Run() })
	for attempt := 0; ; attempt++ {
		if exec.Command("docker", "exec", container, "/garage", "status").Run() == nil {
			break
		}
		if attempt == 100 {
			test.Fatal("garage did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	node, _, _ := strings.Cut(strings.TrimSpace(garage("node", "id", "-q")), "@")
	garage("layout", "assign", "-z", "local", "-c", "1G", node)
	garage("layout", "apply", "--version", "1")
	garage("bucket", "create", "lyra")
	key := garage("key", "create", "lyra-e2e")
	keyID := regexp.MustCompile(`Key ID:\s+(\S+)`).FindStringSubmatch(key)
	secretKey := regexp.MustCompile(`Secret key:\s+(\S+)`).FindStringSubmatch(key)
	if keyID == nil || secretKey == nil {
		test.Fatalf("garage key create output: %s", key)
	}
	garage("bucket", "allow", "--read", "--write", "lyra", "--key", "lyra-e2e")
	objects := func() int {
		test.Helper()
		count, _ := strconv.Atoi(regexp.MustCompile(`Objects:\s+(\d+)`).FindStringSubmatch(garage("bucket", "info", "lyra"))[1])
		return count
	}

	test.Setenv("AWS_ACCESS_KEY_ID", keyID[1])
	test.Setenv("AWS_SECRET_ACCESS_KEY", secretKey[1])
	srv := startServerAt(test, "127.0.0.1:"+freePort(test), filepath.Join(dir, "data"), "127.0.0.1",
		"--storage", "s3", "--s3-endpoint", "http://127.0.0.1:"+s3Port, "--s3-bucket", "lyra")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")

	asset := make([]byte, 12_000_000)
	rand.Read(asset)
	assetPath := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(assetPath, asset, 0o644); err != nil {
		test.Fatal(err)
	}
	releaseJSON(test, srv, token, "create", "acme/demo", "v1.0.0", "--asset", assetPath)
	if count := objects(); count != 1 {
		test.Fatalf("%d objects after the upload, expected 1", count)
	}
	status, data := call(test, http.MethodGet, srv.url+"/acme/demo/releases/latest/download/big.bin", bearer(token), "")
	if status != http.StatusOK || !bytes.Equal(data, asset) {
		test.Fatalf("download from S3: HTTP %d, %d bytes", status, len(data))
	}

	auth := []crane.Option{crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: token})}
	image, err := random.Image(12_000_000, 2)
	if err != nil {
		test.Fatal(err)
	}
	if err := crane.Push(image, srv.address+"/acme/demo:v1.0.0", auth...); err != nil {
		test.Fatalf("push to S3: %v", err)
	}
	pulled, err := crane.Pull(srv.address+"/acme/demo:v1.0.0", auth...)
	if err != nil {
		test.Fatal(err)
	}
	layers, err := pulled.Layers()
	if err != nil {
		test.Fatal(err)
	}
	for _, layer := range layers {
		want, _ := layer.Digest()
		reader, err := layer.Compressed()
		if err != nil {
			test.Fatal(err)
		}
		read, err := io.Copy(io.Discard, reader)
		reader.Close()
		if err != nil || read < 1_000_000 {
			test.Fatalf("layer %s from S3: %d bytes, %v", want, read, err)
		}
	}

	before := objects()
	if result := runLyra(test, srv, token, "release", "delete", "acme/demo", "v1.0.0", "--yes"); result.code != 0 {
		test.Fatalf("release delete: %+v", result)
	}
	if after := objects(); after != before-1 {
		test.Fatalf("objects: %d before deleting the release, %d after; expected one fewer", before, after)
	}
	if _, err := crane.Pull(srv.address+"/acme/demo:v1.0.0", auth...); err != nil {
		test.Fatalf("pull after deleting the release: %v", err)
	}
}
