package repos

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var ErrNotFound = errors.New("not found")

const (
	Verified   = "verified"
	Unverified = "unverified"
	Unsigned   = "unsigned"
)

type Git struct {
	AllowedSigners string
}

type Signature struct {
	Status      string `json:"status"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Principal   string `json:"principal,omitempty"`
	KeyID       int64  `json:"key_id,omitempty"`
	KeyUser     string `json:"key_user,omitempty"`
	KeyAgent    string `json:"key_agent,omitempty"`
}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type Commit struct {
	SHA       string    `json:"sha"`
	Parents   []string  `json:"parents"`
	Author    Person    `json:"author"`
	Committer Person    `json:"committer"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Signature Signature `json:"signature"`
}

type Tag struct {
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	Annotated bool      `json:"annotated"`
	Tagger    *Person   `json:"tagger,omitempty"`
	Message   string    `json:"message,omitempty"`
	Signature Signature `json:"signature"`
}

type Entry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

type Ref struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

var fingerprintPattern = regexp.MustCompile(`key (SHA256:[A-Za-z0-9+/=]+)`)
var principalPattern = regexp.MustCompile(`signature for (\S+) with`)

func (g Git) run(dir string, args ...string) ([]byte, []byte, error) {
	full := append([]string{"-c", "gpg.ssh.allowedSignersFile=" + g.AllowedSigners}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func (g Git) Init(dir string) error {
	cmd := exec.Command("git", "init", "--quiet", "--bare", "--initial-branch=main", dir)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %v: %s", err, output)
	}
	return nil
}

func (g Git) ResolveCommit(dir, rev string) (string, error) {
	if rev == "" || strings.HasPrefix(rev, "-") {
		return "", ErrNotFound
	}
	stdout, _, err := g.run(dir, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return "", ErrNotFound
	}
	return strings.TrimSpace(string(stdout)), nil
}

func (g Git) DefaultBranch(dir string) string {
	stdout, _, err := g.run(dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(stdout))
}

func (g Git) Branches(dir string) ([]Ref, error) {
	stdout, stderr, err := g.run(dir, "for-each-ref", "--format=%(refname:short)%00%(objectname)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("for-each-ref: %v: %s", err, stderr)
	}
	refs := []Ref{}
	for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
		name, sha, found := strings.Cut(line, "\x00")
		if found {
			refs = append(refs, Ref{Name: name, SHA: sha})
		}
	}
	return refs, nil
}

const logFormat = "%H%x00%P%x00%an%x00%ae%x00%aI%x00%cn%x00%ce%x00%cI%x00%G?%x00%GK%x00%GS%x00%s%x00%b%x1e"

func parseCommits(output []byte) []Commit {
	commits := []Commit{}
	for _, record := range strings.Split(string(output), "\x1e") {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x00", 13)
		if len(fields) < 13 {
			continue
		}
		commit := Commit{
			SHA:       fields[0],
			Parents:   strings.Fields(fields[1]),
			Author:    Person{Name: fields[2], Email: fields[3], Date: fields[4]},
			Committer: Person{Name: fields[5], Email: fields[6], Date: fields[7]},
			Subject:   fields[11],
			Body:      strings.TrimSpace(fields[12]),
		}
		switch fields[8] {
		case "G":
			commit.Signature = Signature{Status: Verified, Fingerprint: fields[9], Principal: fields[10]}
		case "N":
			commit.Signature = Signature{Status: Unsigned}
		default:
			commit.Signature = Signature{Status: Unverified, Fingerprint: fields[9]}
		}
		commits = append(commits, commit)
	}
	return commits
}

func (g Git) Log(dir, rev string, limit int) ([]Commit, error) {
	sha, err := g.ResolveCommit(dir, rev)
	if err != nil {
		return nil, err
	}
	stdout, stderr, err := g.run(dir, "log", "--max-count="+strconv.Itoa(limit), "--format="+logFormat, sha, "--")
	if err != nil {
		return nil, fmt.Errorf("git log: %v: %s", err, stderr)
	}
	return parseCommits(stdout), nil
}

func verifyOutput(stderr []byte, err error) Signature {
	output := string(stderr)
	signature := Signature{Status: Unverified}
	if match := fingerprintPattern.FindStringSubmatch(output); match != nil {
		signature.Fingerprint = match[1]
	}
	if err == nil {
		signature.Status = Verified
		if match := principalPattern.FindStringSubmatch(output); match != nil {
			signature.Principal = match[1]
		}
	}
	return signature
}

func (g Git) Commit(dir, rev string) (Commit, error) {
	sha, err := g.ResolveCommit(dir, rev)
	if err != nil {
		return Commit{}, err
	}
	stdout, stderr, err := g.run(dir, "log", "--max-count=1", "--format="+logFormat, sha, "--")
	if err != nil {
		return Commit{}, fmt.Errorf("git log: %v: %s", err, stderr)
	}
	commits := parseCommits(stdout)
	if len(commits) != 1 {
		return Commit{}, ErrNotFound
	}
	commit := commits[0]
	raw, _, err := g.run(dir, "cat-file", "commit", sha)
	if err != nil {
		return Commit{}, err
	}
	header, _, _ := strings.Cut(string(raw), "\n\n")
	if !strings.Contains("\n"+header, "\ngpgsig") {
		commit.Signature = Signature{Status: Unsigned}
		return commit, nil
	}
	_, verifyStderr, verifyErr := g.run(dir, "verify-commit", sha)
	commit.Signature = verifyOutput(verifyStderr, verifyErr)
	return commit, nil
}

func (g Git) Patch(dir, sha string) (string, error) {
	stdout, stderr, err := g.run(dir, "show", "--format=", "--stat", "--patch", "--no-color", sha, "--")
	if err != nil {
		return "", fmt.Errorf("git show: %v: %s", err, stderr)
	}
	return string(stdout), nil
}

func (g Git) Tags(dir string) ([]Tag, error) {
	format := "%(refname:short)%00%(objecttype)%00%(*objectname)%00%(objectname)%00%(taggername)%00%(taggeremail:trim)%00%(taggerdate:iso-strict)%00%(contents:subject)%00%(contents:signature)%1e"
	stdout, stderr, err := g.run(dir, "for-each-ref", "--sort=-creatordate", "--format="+format, "refs/tags")
	if err != nil {
		return nil, fmt.Errorf("for-each-ref: %v: %s", err, stderr)
	}
	tags := []Tag{}
	for _, record := range strings.Split(string(stdout), "\x1e") {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x00", 9)
		if len(fields) < 9 {
			continue
		}
		tag := Tag{Name: fields[0], Target: fields[3], Signature: Signature{Status: Unsigned}}
		if fields[1] == "tag" {
			tag.Annotated = true
			tag.Target = fields[2]
			tag.Tagger = &Person{Name: fields[4], Email: fields[5], Date: fields[6]}
			tag.Message = fields[7]
			if strings.TrimSpace(fields[8]) != "" {
				_, verifyStderr, verifyErr := g.run(dir, "verify-tag", "--end-of-options", "refs/tags/"+tag.Name)
				tag.Signature = verifyOutput(verifyStderr, verifyErr)
			}
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (g Git) Tree(dir, sha, path string) ([]Entry, error) {
	object := sha + "^{tree}"
	if path != "" {
		object = sha + ":" + path
	}
	kind, _, err := g.run(dir, "cat-file", "-t", object)
	if err != nil || strings.TrimSpace(string(kind)) != "tree" {
		return nil, ErrNotFound
	}
	stdout, stderr, err := g.run(dir, "ls-tree", "-z", "--long", object)
	if err != nil {
		return nil, fmt.Errorf("ls-tree: %v: %s", err, stderr)
	}
	entries := []Entry{}
	for _, line := range strings.Split(string(stdout), "\x00") {
		meta, name, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		parts := strings.Fields(meta)
		if len(parts) < 4 {
			continue
		}
		size, _ := strconv.ParseInt(parts[3], 10, 64)
		entries = append(entries, Entry{Name: name, Mode: parts[0], Type: parts[1], SHA: parts[2], Size: size})
	}
	return entries, nil
}

func (g Git) Blob(dir, sha, path string) ([]byte, error) {
	object := sha + ":" + path
	kind, _, err := g.run(dir, "cat-file", "-t", object)
	if err != nil || strings.TrimSpace(string(kind)) != "blob" {
		return nil, ErrNotFound
	}
	stdout, stderr, err := g.run(dir, "cat-file", "blob", object)
	if err != nil {
		return nil, fmt.Errorf("cat-file: %v: %s", err, stderr)
	}
	return stdout, nil
}
