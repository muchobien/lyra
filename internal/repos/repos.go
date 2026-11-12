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

type Git struct{}

type Signature struct {
	Status      string `json:"status"`
	Fingerprint string `json:"fingerprint,omitempty"`
	KeyID       int64  `json:"key_id,omitempty"`
	KeyUser     string `json:"key_user,omitempty"`
	KeyAgent    string `json:"key_agent,omitempty"`
	Payload     []byte `json:"-"`
	Armored     []byte `json:"-"`
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

var signatureStart = regexp.MustCompile(`(?m)^-----BEGIN (PGP|SSH) SIGNATURE-----$|^-----BEGIN SIGNED MESSAGE-----$`)

func (g Git) command(dir string, stdin []byte, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func (g Git) run(dir string, args ...string) ([]byte, []byte, error) {
	return g.command(dir, nil, args...)
}

func (g Git) objects(dir string, shas []string) (map[string][]byte, error) {
	objects := map[string][]byte{}
	if len(shas) == 0 {
		return objects, nil
	}
	stdout, stderr, err := g.command(dir, []byte(strings.Join(shas, "\n")+"\n"), "cat-file", "--batch")
	if err != nil {
		return nil, fmt.Errorf("cat-file --batch: %v: %s", err, stderr)
	}
	for len(stdout) > 0 {
		header, rest, found := bytes.Cut(stdout, []byte("\n"))
		if !found {
			break
		}
		fields := strings.Fields(string(header))
		if len(fields) != 3 {
			return nil, fmt.Errorf("cat-file --batch: unexpected header %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size+1 > len(rest) {
			return nil, fmt.Errorf("cat-file --batch: bad size in %q", header)
		}
		objects[fields[0]] = rest[:size]
		stdout = rest[size+1:]
	}
	return objects, nil
}

func commitSignature(raw []byte) Signature {
	headers, message := raw, []byte(nil)
	if end := bytes.Index(raw, []byte("\n\n")); end >= 0 {
		headers, message = raw[:end+1], raw[end+1:]
	}
	var payload, armored bytes.Buffer
	inSignature := false
	for _, line := range bytes.SplitAfter(headers, []byte("\n")) {
		if inSignature && bytes.HasPrefix(line, []byte(" ")) {
			armored.Write(line[1:])
			continue
		}
		inSignature = false
		if value, isSignature := bytes.CutPrefix(line, []byte("gpgsig ")); isSignature {
			inSignature = true
			armored.Write(value)
			continue
		}
		if value, isSignature := bytes.CutPrefix(line, []byte("gpgsig-sha256 ")); isSignature {
			inSignature = true
			armored.Write(value)
			continue
		}
		payload.Write(line)
	}
	if armored.Len() == 0 {
		return Signature{Status: Unsigned}
	}
	payload.Write(message)
	return Signature{Status: Unverified, Payload: payload.Bytes(), Armored: armored.Bytes()}
}

func tagSignature(raw []byte) Signature {
	matches := signatureStart.FindAllIndex(raw, -1)
	if matches == nil {
		return Signature{Status: Unsigned}
	}
	start := matches[len(matches)-1][0]
	return Signature{Status: Unverified, Payload: raw[:start], Armored: raw[start:]}
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

const logFormat = "%H%x00%P%x00%an%x00%ae%x00%aI%x00%cn%x00%ce%x00%cI%x00%s%x00%b%x1e"

func (g Git) Log(dir, rev string, limit int, exclude ...string) ([]Commit, error) {
	sha, err := g.ResolveCommit(dir, rev)
	if err != nil {
		return nil, err
	}
	args := []string{"log", "--max-count=" + strconv.Itoa(limit), "--format=" + logFormat, sha}
	for _, excluded := range exclude {
		args = append(args, "^"+excluded)
	}
	stdout, stderr, err := g.run(dir, append(args, "--")...)
	if err != nil {
		return nil, fmt.Errorf("git log: %v: %s", err, stderr)
	}
	commits := []Commit{}
	shas := []string{}
	for _, record := range strings.Split(string(stdout), "\x1e") {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x00", 10)
		if len(fields) < 10 {
			continue
		}
		commits = append(commits, Commit{
			SHA:       fields[0],
			Parents:   strings.Fields(fields[1]),
			Author:    Person{Name: fields[2], Email: fields[3], Date: fields[4]},
			Committer: Person{Name: fields[5], Email: fields[6], Date: fields[7]},
			Subject:   fields[8],
			Body:      strings.TrimSpace(fields[9]),
		})
		shas = append(shas, fields[0])
	}
	objects, err := g.objects(dir, shas)
	if err != nil {
		return nil, err
	}
	for index := range commits {
		commits[index].Signature = commitSignature(objects[commits[index].SHA])
	}
	return commits, nil
}

func (g Git) Commit(dir, rev string) (Commit, error) {
	commits, err := g.Log(dir, rev, 1)
	if err != nil {
		return Commit{}, err
	}
	if len(commits) != 1 {
		return Commit{}, ErrNotFound
	}
	return commits[0], nil
}

func (g Git) Patch(dir, sha string) (string, error) {
	stdout, stderr, err := g.run(dir, "show", "--format=", "--stat", "--patch", "--no-color", sha, "--")
	if err != nil {
		return "", fmt.Errorf("git show: %v: %s", err, stderr)
	}
	return string(stdout), nil
}

func (g Git) MergeBase(dir, first, second string) (string, error) {
	stdout, _, err := g.run(dir, "merge-base", first, second)
	if err != nil {
		return "", ErrNotFound
	}
	return strings.TrimSpace(string(stdout)), nil
}

func (g Git) IsAncestor(dir, ancestor, descendant string) (bool, error) {
	_, stderr, err := g.run(dir, "merge-base", "--is-ancestor", ancestor, descendant)
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("merge-base --is-ancestor: %v: %s", err, stderr)
	}
	return true, nil
}

func (g Git) UpdateRef(dir, ref, sha string) error {
	if _, stderr, err := g.run(dir, "update-ref", ref, sha); err != nil {
		return fmt.Errorf("update-ref %s: %v: %s", ref, err, stderr)
	}
	return nil
}

func (g Git) Diff(dir, base, head string) (string, error) {
	stdout, stderr, err := g.run(dir, "diff", "--no-color", "--find-renames", base, head, "--")
	if err != nil {
		return "", fmt.Errorf("git diff: %v: %s", err, stderr)
	}
	return string(stdout), nil
}

type FileChange struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary,omitempty"`
}

var fileStatuses = map[byte]string{'A': "added", 'D': "deleted", 'M': "modified", 'R': "renamed", 'C': "copied", 'T': "type-changed"}

func (g Git) DiffFiles(dir, base, head string) ([]FileChange, error) {
	stdout, stderr, err := g.run(dir, "diff", "--find-renames", "--name-status", "-z", base, head, "--")
	if err != nil {
		return nil, fmt.Errorf("git diff --name-status: %v: %s", err, stderr)
	}
	files := []FileChange{}
	fields := strings.Split(strings.TrimSuffix(string(stdout), "\x00"), "\x00")
	for index := 0; index+1 < len(fields); index += 2 {
		change := FileChange{Status: fileStatuses[fields[index][0]], Path: fields[index+1]}
		if fields[index][0] == 'R' || fields[index][0] == 'C' {
			change.OldPath = fields[index+1]
			change.Path = fields[index+2]
			index++
		}
		files = append(files, change)
	}
	stdout, stderr, err = g.run(dir, "diff", "--find-renames", "--numstat", "-z", base, head, "--")
	if err != nil {
		return nil, fmt.Errorf("git diff --numstat: %v: %s", err, stderr)
	}
	counts := strings.Split(strings.TrimSuffix(string(stdout), "\x00"), "\x00")
	for index, file := 0, 0; index < len(counts) && file < len(files); file++ {
		numbers := strings.Split(counts[index], "\t")
		index++
		if len(numbers) == 3 && numbers[2] == "" {
			index += 2
		}
		if len(numbers) < 2 {
			continue
		}
		if numbers[0] == "-" {
			files[file].Binary = true
			continue
		}
		files[file].Additions, _ = strconv.Atoi(numbers[0])
		files[file].Deletions, _ = strconv.Atoi(numbers[1])
	}
	return files, nil
}

func (g Git) Tags(dir string) ([]Tag, error) {
	format := "%(refname:short)%00%(objecttype)%00%(*objectname)%00%(objectname)%00%(taggername)%00%(taggeremail:trim)%00%(taggerdate:iso-strict)%00%(contents:subject)%1e"
	stdout, stderr, err := g.run(dir, "for-each-ref", "--sort=-creatordate", "--format="+format, "refs/tags")
	if err != nil {
		return nil, fmt.Errorf("for-each-ref: %v: %s", err, stderr)
	}
	tags := []Tag{}
	tagObjects := []string{}
	for _, record := range strings.Split(string(stdout), "\x1e") {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x00", 8)
		if len(fields) < 8 {
			continue
		}
		tag := Tag{Name: fields[0], Target: fields[3], Signature: Signature{Status: Unsigned}}
		if fields[1] == "tag" {
			tag.Annotated = true
			tag.Target = fields[2]
			tag.Tagger = &Person{Name: fields[4], Email: fields[5], Date: fields[6]}
			tag.Message = fields[7]
			tagObjects = append(tagObjects, fields[3])
		}
		tags = append(tags, tag)
	}
	objects, err := g.objects(dir, tagObjects)
	if err != nil {
		return nil, err
	}
	objectIndex := 0
	for index := range tags {
		if tags[index].Annotated {
			tags[index].Signature = tagSignature(objects[tagObjects[objectIndex]])
			objectIndex++
		}
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
