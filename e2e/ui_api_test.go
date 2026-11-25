package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedUI(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	status, body := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/tree", tailnet(), "")
	var tree struct {
		Empty   bool `json:"empty"`
		Entries []struct {
			Name string `json:"name"`
		} `json:"entries"`
		Readme string `json:"readme"`
	}
	if err := json.Unmarshal(body, &tree); err != nil || status != http.StatusOK || !tree.Empty || len(tree.Entries) != 0 {
		test.Fatalf("empty tree: HTTP %d, %s, %v", status, body, err)
	}
	src := newSource(test)
	if err := os.MkdirAll(filepath.Join(src, "src"), 0o750); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, "README.md", "# Demo\n\n<script>alert('unsafe')</script>\n", "Document the project")
	commitFile(test, src, "src/example.go", "package example\n\n// <script>alert('unsafe')</script>\n", "Add source")
	commitFile(test, src, "data.bin", "\x00\xff", "Add binary")
	commitFile(test, src, "large.txt", strings.Repeat("x", 1<<20+1), "Add large file")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/tree", tailnet(), "")
	if err := json.Unmarshal(body, &tree); err != nil || status != http.StatusOK || tree.Empty || len(tree.Entries) != 4 {
		test.Fatalf("populated tree: HTTP %d, %s, %v", status, body, err)
	}
	if !strings.Contains(tree.Readme, "<h1>Demo</h1>") || strings.Contains(tree.Readme, "<script") {
		test.Fatalf("unsafe or missing Markdown rendering: %s", tree.Readme)
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/files", tailnet(), "")
	var files []string
	if err := json.Unmarshal(body, &files); err != nil || status != http.StatusOK || !strings.Contains(strings.Join(files, "|"), "src/example.go") {
		test.Fatalf("recursive files: HTTP %d, %s, %v", status, body, err)
	}
	var blob struct {
		Code     string `json:"code"`
		Markdown string `json:"markdown"`
		Binary   bool   `json:"binary"`
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blob?path=src/example.go", tailnet(), "")
	if err := json.Unmarshal(body, &blob); err != nil || status != http.StatusOK || blob.Binary || !strings.Contains(blob.Code, `class="chroma"`) || strings.Contains(blob.Code, "<script") {
		test.Fatalf("highlighted blob: HTTP %d, %s, %v", status, body, err)
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blob?path=README.md", tailnet(), "")
	if err := json.Unmarshal(body, &blob); err != nil || status != http.StatusOK || !strings.Contains(blob.Code, `class="chroma"`) || !strings.Contains(blob.Markdown, "<h1>Demo</h1>") || strings.Contains(blob.Markdown, "<script") {
		test.Fatalf("rendered Markdown blob: HTTP %d, %s, %v", status, body, err)
	}
	for _, path := range []string{"data.bin", "large.txt"} {
		status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blob?path="+path, tailnet(), "")
		if err := json.Unmarshal(body, &blob); err != nil || status != http.StatusOK || !blob.Binary || blob.Code != "" || blob.Markdown != "" {
			test.Fatalf("unrendered file %s: HTTP %d, %s, %v", path, status, body, err)
		}
	}
	request, _ := http.NewRequest(http.MethodGet, srv.url+"/api/repos/acme/demo/raw?path=data.bin", nil)
	request.Header.Set("Tailscale-User-Login", login)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	raw, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || string(raw) != "\x00\xff" || response.Header.Get("Content-Security-Policy") != "sandbox" || response.Header.Get("X-Content-Type-Options") != "nosniff" {
		test.Fatalf("raw download: HTTP %d, headers %v", response.StatusCode, response.Header)
	}
	mustGit(test, src, "checkout", "-q", "-b", "feature/nested")
	commitFile(test, src, "branch.txt", "branch content\n", "Branch commit")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "feature/nested")
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/raw?ref="+url.QueryEscape("feature/nested")+"&path=branch.txt", tailnet(), "")
	if status != http.StatusOK || string(body) != "branch content\n" {
		test.Fatalf("branch with slash: HTTP %d, %s", status, body)
	}
	reader := createToken(test, srv, `{"name":"other","scope":"read","repos":["other/*"]}`)
	for _, path := range []string{"tree", "files", "blob?path=README.md"} {
		if status, _ := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/"+path, bearer(reader), ""); status != http.StatusForbidden {
			test.Fatalf("unauthorized %s: HTTP %d", path, status)
		}
	}
	for _, path := range []string{"tree?path=missing", "tree?ref=missing", "blob?path=missing"} {
		if status, _ := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/"+path, tailnet(), ""); status != http.StatusNotFound {
			test.Fatalf("missing %s: HTTP %d", path, status)
		}
	}
	status, rawFile := call(test, http.MethodGet, srv.url+"/acme/demo/blob/feature/nested/branch.txt?raw", tailnet(), "")
	if status != http.StatusOK || string(rawFile) != "branch content\n" {
		test.Fatalf("raw file by path: HTTP %d, %s", status, rawFile)
	}
	for _, target := range []string{"/acme/demo/blob/feature/nested/branch.txt", "/acme/demo/blob/refs/heads/feature/nested/branch.txt", "/acme/demo/tree/main", "/acme/demo/tree/main/src", "/acme/demo/blob/main/src/example.go"} {
		status, body = call(test, http.MethodGet, srv.url+target, tailnet(), "")
		if status != http.StatusOK || !strings.Contains(string(body), `<div id="root"></div>`) {
			test.Fatalf("UI deep link %s: HTTP %d, %s", target, status, body)
		}
	}
	asset := regexp.MustCompile(`src="(/static/assets/[^"]+\.js)"`).FindSubmatch(body)
	if len(asset) != 2 {
		test.Fatalf("no embedded script in shell: %s", body)
	}
	status, body = call(test, http.MethodGet, srv.url+string(asset[1]), tailnet(), "")
	if status != http.StatusOK || len(body) < 1000 || strings.Contains(string(body), `<div id="root">`) {
		test.Fatalf("embedded script: HTTP %d, size %d", status, len(body))
	}
	for _, target := range []string{"/static/assets/missing.js", "/static/assets/"} {
		request, _ := http.NewRequest(http.MethodGet, srv.url+target, nil)
		request.Header.Set("Tailscale-User-Login", login)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			test.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound || strings.Contains(response.Header.Get("Cache-Control"), "immutable") {
			test.Fatalf("missing asset %s: HTTP %d, Cache-Control %q", target, response.StatusCode, response.Header.Get("Cache-Control"))
		}
	}
	request, _ = http.NewRequest(http.MethodGet, srv.url+"/api/code.css", nil)
	request.Header.Set("Tailscale-User-Login", login)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	styles, _ := io.ReadAll(response.Body)
	response.Body.Close()
	tag := response.Header.Get("ETag")
	if response.StatusCode != http.StatusOK || !strings.Contains(string(styles), ":root[data-theme=dark]") || tag == "" {
		test.Fatalf("highlight theme styles: HTTP %d, ETag %q", response.StatusCode, tag)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/api/code.css", map[string]string{"Tailscale-User-Login": login, "If-None-Match": tag}, ""); status != http.StatusNotModified {
		test.Fatalf("revalidated theme styles: HTTP %d", status)
	}
	for _, target := range []string{"pulls", "issues", "runs", "releases", "images", "activity", "settings", "tags", "commits/main"} {
		status, body := call(test, http.MethodGet, srv.url+"/acme/demo/"+target, tailnet(), "")
		if status != http.StatusOK || !strings.Contains(string(body), `<div id="root"></div>`) {
			test.Fatalf("workspace shell %s: HTTP %d", target, status)
		}
	}
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for target, location := range map[string]string{
		"/acme/demo.git":                     "/acme/demo",
		"/acme/demo.git/":                    "/acme/demo/",
		"/acme/demo.git/tree/main/src?plain": "/acme/demo/tree/main/src?plain",
		"/acme/demo.git/pulls/1":             "/acme/demo/pulls/1",
	} {
		request, _ := http.NewRequest(http.MethodGet, srv.url+target, nil)
		request.Header.Set("Tailscale-User-Login", login)
		response, err := noRedirect.Do(request)
		if err != nil {
			test.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusMovedPermanently || response.Header.Get("Location") != location {
			test.Fatalf("redirect %s: HTTP %d, Location %q", target, response.StatusCode, response.Header.Get("Location"))
		}
	}
	status, body = call(test, http.MethodGet, srv.url+"/acme/demo.git", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(string(body), `<div id="root"></div>`) {
		test.Fatalf("followed .git redirect: HTTP %d", status)
	}
	for _, target := range []string{"/acme/missing", "/acme/demo/issues/100", "/acme/demo/pulls/100", "/acme/demo/runs/100", "/acme/demo/commit/missing", "/acme/demo/commits/missing", "/acme/demo/releases/tag/missing", "/acme/demo/tree/missing", "/acme/demo/tree/main/missing", "/acme/demo/blob/main/missing", "/acme/demo/blob/feature/branch.txt", "/acme/demo/missing", "/acme/demo/tree", "/acme/demo/one/two/three", "/acme/demo.git/missing"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			status, body := call(test, method, srv.url+target, tailnet(), "")
			if status != http.StatusNotFound || (method == http.MethodHead && len(body) != 0) {
				test.Fatalf("missing workspace %s %s: HTTP %d", method, target, status)
			}
			if method == http.MethodGet && !strings.Contains(string(body), `data-error-title="Not Found"`) {
				test.Fatalf("error shell lacks its status: %s", target)
			}
		}
	}
	status, body = call(test, http.MethodGet, srv.url+"/acme/demo", bearer(reader), "")
	if status != http.StatusForbidden || !strings.Contains(string(body), `data-error-title="Forbidden"`) {
		test.Fatalf("forbidden workspace: HTTP %d", status)
	}
}
