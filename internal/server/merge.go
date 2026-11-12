package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hiddeco/sshsig"
	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/store"
)

type conflictError string

func (conflict conflictError) Error() string {
	return string(conflict)
}

type mergePlan struct {
	HeadSHA     string   `json:"head_sha"`
	BaseSHA     string   `json:"base_sha"`
	FastForward bool     `json:"fast_forward"`
	Conflicts   []string `json:"conflicts"`
	Methods     []string `json:"methods"`
	Blocked     []string `json:"blocked"`
	tree        string
}

type mergeRequest struct {
	Method       string `json:"method"`
	HeadSHA      string `json:"head_sha"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	DeleteBranch bool   `json:"delete_branch"`
}

func (s *Server) planMerge(repo string, pull store.Pull) (mergePlan, error) {
	dir := s.repoDir(repo)
	plan := mergePlan{Blocked: []string{}}
	var err error
	if plan.HeadSHA, err = s.git.ResolveCommit(dir, "refs/heads/"+pull.Head); err != nil {
		return plan, store.InvalidError("branch " + pull.Head + " no longer exists")
	}
	if plan.BaseSHA, err = s.git.ResolveCommit(dir, "refs/heads/"+pull.Base); err != nil {
		return plan, store.InvalidError("branch " + pull.Base + " no longer exists")
	}
	if plan.FastForward, err = s.git.IsAncestor(dir, plan.BaseSHA, plan.HeadSHA); err != nil {
		return plan, err
	}
	if plan.tree, plan.Conflicts, err = s.git.MergeTree(dir, plan.BaseSHA, plan.HeadSHA); err != nil {
		return plan, err
	}
	rules, err := s.store.Rules(repo)
	if err != nil {
		return plan, err
	}
	policy := policyFor(rules, pull.Base)
	plan.Methods = policy.methods
	if policy.requireApproval {
		reviews, err := s.store.Reviews(pull.Item)
		if err != nil {
			return plan, err
		}
		for _, problem := range approvalProblems(pull, reviews, plan.HeadSHA) {
			plan.Blocked = append(plan.Blocked, "branch rule "+strings.Join(policy.rules, ", ")+": "+problem)
		}
	}
	if len(policy.methods) == 0 {
		plan.Blocked = append(plan.Blocked, "branch rules "+strings.Join(policy.rules, ", ")+" allow no merge method")
	}
	if policy.requireSigned {
		commits, err := s.git.Log(dir, plan.HeadSHA, 10000, plan.BaseSHA)
		if err != nil {
			return plan, err
		}
		keys, err := s.store.Keys("")
		if err != nil {
			return plan, err
		}
		problems, err := unverifiedCommits(keys, commits)
		if err != nil {
			return plan, err
		}
		for _, problem := range problems {
			plan.Blocked = append(plan.Blocked, "branch rule "+strings.Join(policy.rules, ", ")+" requires verified signatures: "+problem)
		}
	}
	return plan, nil
}

func gitIdentity(name, email string, when time.Time) string {
	return fmt.Sprintf("%s <%s> %d +0000", name, email, when.Unix())
}

func (s *Server) mergePull(r *http.Request, repo string, number int64, request mergeRequest) (store.Pull, int64, error) {
	id := identityFrom(r)
	dir := s.repoDir(repo)
	pull, err := s.store.Pull(repo, number)
	if err != nil {
		return pull, 0, err
	}
	if pull.State != "open" {
		return pull, 0, store.InvalidError(fmt.Sprintf("pull request #%d is %s", number, pull.State))
	}
	if request.Method != "ff" && request.Method != "squash" {
		return pull, 0, store.InvalidError("method must be ff or squash")
	}
	plan, err := s.planMerge(repo, pull)
	if err != nil {
		return pull, 0, err
	}
	if len(plan.Blocked) > 0 {
		return pull, 0, conflictError(strings.Join(plan.Blocked, "; "))
	}
	if !slices.Contains(plan.Methods, request.Method) {
		return pull, 0, conflictError("branch rules for " + pull.Base + " allow only " + strings.Join(plan.Methods, ", "))
	}
	if request.HeadSHA != "" && request.HeadSHA != plan.HeadSHA {
		return pull, 0, conflictError(fmt.Sprintf("%s is at %s, not %s", pull.Head, plan.HeadSHA, request.HeadSHA))
	}
	if plan.HeadSHA != pull.HeadSHA {
		if err := s.git.UpdateRef(dir, fmt.Sprintf("refs/pull/%d/head", number), plan.HeadSHA, ""); err != nil {
			return pull, 0, err
		}
		if _, err := s.store.SyncPull(repo, number, id.actor, plan.HeadSHA, pull.BaseSHA); err != nil {
			return pull, 0, err
		}
	}

	mergeSHA := plan.HeadSHA
	switch {
	case request.Method == "ff" && !plan.FastForward:
		return pull, 0, conflictError(pull.Base + " has commits that " + pull.Head + " doesn't; rebase it or merge with squash")
	case request.Method == "squash" && len(plan.Conflicts) > 0:
		return pull, 0, conflictError("merging " + pull.Head + " into " + pull.Base + " conflicts in " + strings.Join(plan.Conflicts, ", "))
	case request.Method == "squash":
		commits, err := s.git.Log(dir, plan.HeadSHA, 1000, plan.BaseSHA)
		if err != nil {
			return pull, 0, err
		}
		if len(commits) == 0 {
			return pull, 0, store.InvalidError(pull.Base + " already contains every commit on " + pull.Head)
		}
		author := commits[len(commits)-1].Author
		title := strings.TrimSpace(request.Title)
		if title == "" {
			title = fmt.Sprintf("%s (#%d)", pull.Title, number)
		}
		message := strings.TrimSpace(request.Message)
		if request.Message == "" {
			message = strings.TrimSpace(pull.Body)
		}
		coauthors := []string{}
		for index := len(commits) - 1; index >= 0; index-- {
			trailer := fmt.Sprintf("Co-authored-by: %s <%s>", commits[index].Author.Name, commits[index].Author.Email)
			if commits[index].Author.Email != author.Email && !slices.Contains(coauthors, trailer) {
				coauthors = append(coauthors, trailer)
			}
		}
		text := title + "\n"
		if message != "" {
			text += "\n" + message + "\n"
		}
		if len(coauthors) > 0 {
			text += "\n" + strings.Join(coauthors, "\n") + "\n"
		}
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		now := time.Now()
		headers := "tree " + plan.tree + "\nparent " + plan.BaseSHA + "\nauthor " + gitIdentity(author.Name, author.Email, now) + "\ncommitter " + gitIdentity("Lyra", "lyra@"+host, now) + "\n"
		signature, err := sshsig.Sign(strings.NewReader(headers+"\n"+text), s.signer, sshsig.HashSHA512, "git")
		if err != nil {
			return pull, 0, err
		}
		armored := strings.TrimSuffix(string(sshsig.Armor(signature)), "\n")
		raw := headers + "gpgsig " + strings.ReplaceAll(armored, "\n", "\n ") + "\n\n" + text
		if mergeSHA, err = s.git.WriteCommit(dir, []byte(raw)); err != nil {
			return pull, 0, err
		}
	}

	if err := s.git.UpdateRef(dir, "refs/heads/"+pull.Base, mergeSHA, plan.BaseSHA); err != nil {
		return pull, 0, conflictError(pull.Base + " moved while merging; try again")
	}
	mergeBase, err := s.git.MergeBase(dir, plan.BaseSHA, plan.HeadSHA)
	if err != nil {
		mergeBase = pull.BaseSHA
	}
	eventID, err := s.store.MarkPullMerged(repo, number, id.actor, mergeSHA, mergeBase, request.Method)
	if err != nil {
		return pull, 0, err
	}
	updates := []RefUpdate{{Ref: "refs/heads/" + pull.Base, Old: plan.BaseSHA, New: mergeSHA}}
	if request.DeleteBranch {
		if err := s.git.UpdateRef(dir, "refs/heads/"+pull.Head, "", plan.HeadSHA); err == nil {
			updates = append(updates, RefUpdate{Ref: "refs/heads/" + pull.Head, Old: plan.HeadSHA, New: strings.Repeat("0", len(plan.HeadSHA))})
		}
	}
	if _, err := s.store.AddEvent("push", repo, id.actor, map[string]any{"refs": updates, "pull": number, "method": request.Method}); err != nil {
		return pull, eventID, err
	}
	if pull.Base == s.git.DefaultBranch(dir) {
		if _, err := closeReferenced(s.store, repo, id.actor, pull.Title+"\n"+pull.Body, mergeSHA); err != nil {
			return pull, eventID, err
		}
	}
	if _, err := ProcessPush(s.store, s.git, dir, repo, id.actor, updates); err != nil {
		return pull, eventID, err
	}
	pull, err = s.store.Pull(repo, number)
	return pull, eventID, err
}

func (s *Server) handleMergePull(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	number, ok := itemNumber(w, r)
	if !ok {
		return
	}
	var request mergeRequest
	if !decodeBody(w, r, &request) {
		return
	}
	if problem := checkText(nil, &request.Message); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	pull, eventID, err := s.mergePull(r, repo.Name, number, request)
	var conflict conflictError
	if errors.As(err, &conflict) {
		writeError(w, http.StatusConflict, conflict.Error())
		return
	}
	if err != nil {
		writeStoreError(w, err, "pull request not found")
		return
	}
	writeJSON(w, http.StatusOK, pullWrite{Pull: pull, EventID: eventID})
}

func (s *Server) handleSigningKey(w http.ResponseWriter, r *http.Request) {
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.signer.PublicKey())))
	writeJSON(w, http.StatusOK, map[string]string{
		"public_key":      publicKey,
		"fingerprint":     ssh.FingerprintSHA256(s.signer.PublicKey()),
		"allowed_signers": `lyra namespaces="git" ` + publicKey,
	})
}

func (s *Server) uiMergePull(w http.ResponseWriter, r *http.Request) {
	repo, item, ok := s.uiItem(w, r, "pr", true)
	if !ok {
		return
	}
	request := mergeRequest{Method: r.FormValue("method"), HeadSHA: r.FormValue("head_sha"), DeleteBranch: r.FormValue("delete_branch") != ""}
	_, _, err := s.mergePull(r, repo.Name, item.Number, request)
	var conflict conflictError
	if errors.As(err, &conflict) {
		s.renderError(w, r, http.StatusConflict, conflict.Error())
		return
	}
	if err != nil {
		s.renderStoreError(w, r, err, "Pull request not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/pulls/%d", repo.Name, item.Number), http.StatusSeeOther)
}
