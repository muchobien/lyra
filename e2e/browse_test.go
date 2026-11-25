package e2e

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func pushDemo(test *testing.T, srv server, token string) string {
	test.Helper()
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "Document the project")
	mustGit(test, src, "checkout", "-q", "-b", "feature/nested")
	commitFile(test, src, "branch.txt", "branch content\n", "Branch commit")
	mustGit(test, src, "checkout", "-q", "main")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main", "feature/nested")
	return src
}

func TestGoImport(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	pushDemo(test, srv, token)
	host := strings.TrimPrefix(srv.url, "http://")
	meta := `<meta name="go-import" content="` + host + `/acme/demo git ` + srv.url + `/acme/demo.git">`
	for _, target := range []string{"/acme/demo", "/acme/demo.git", "/acme/demo/internal/pkg", "/acme/demo/tree/main"} {
		status, body := call(test, http.MethodGet, srv.url+target+"?go-get=1", bearer(token), "")
		if status != http.StatusOK || !strings.Contains(string(body), meta) {
			test.Fatalf("go-import for %s: HTTP %d, %s", target, status, body)
		}
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo?go-get=1", nil, ""); status != http.StatusUnauthorized {
		test.Fatalf("go-import without credentials: HTTP %d, %s", status, body)
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo/internal/pkg", bearer(token), ""); status != http.StatusNotFound || strings.Contains(string(body), "go-import") {
		test.Fatalf("package path without go-get: HTTP %d", status)
	}
}

func TestArchives(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	pushDemo(test, srv, token)

	status, body := call(test, http.MethodGet, srv.url+"/acme/demo/archive/feature/nested.tar.gz", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("tar.gz archive: HTTP %d, %s", status, body)
	}
	unzipped, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		test.Fatal(err)
	}
	names := []string{}
	reader := tar.NewReader(unzipped)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			test.Fatal(err)
		}
		names = append(names, header.Name)
	}
	if !slices.Contains(names, "demo-feature-nested/branch.txt") || !slices.Contains(names, "demo-feature-nested/README.md") {
		test.Fatalf("tar.gz entries: %v", names)
	}

	request, _ := http.NewRequest(http.MethodGet, srv.url+"/acme/demo/archive/main.zip", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Disposition") != `attachment; filename=demo-main.zip` {
		test.Fatalf("zip archive: HTTP %d, headers %v", response.StatusCode, response.Header)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		test.Fatal(err)
	}
	names = []string{}
	for _, entry := range archive.File {
		names = append(names, entry.Name)
	}
	if !slices.Contains(names, "demo-main/README.md") || slices.Contains(names, "demo-main/branch.txt") {
		test.Fatalf("zip entries: %v", names)
	}

	for _, target := range []string{"/acme/demo/archive/missing.zip", "/acme/demo/archive/main.rar", "/acme/missing/archive/main.zip"} {
		if status, body := call(test, http.MethodGet, srv.url+target, bearer(token), ""); status != http.StatusNotFound {
			test.Fatalf("%s: HTTP %d, %s", target, status, body)
		}
	}
	outsider := createToken(test, srv, `{"name":"other","scope":"read","repos":["other/*"]}`)
	if status, _ := call(test, http.MethodGet, srv.url+"/acme/demo/archive/main.zip", bearer(outsider), ""); status != http.StatusForbidden {
		test.Fatalf("archive without access: HTTP %d", status)
	}
}
