package server

import (
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var mergeMethods = []string{"ff", "squash"}

type branchPolicy struct {
	rules           []string
	requireSigned   bool
	requireApproval bool
	requireChecks   bool
	forbidForcePush bool
	forbidDeletion  bool
	methods         []string
}

func policyFor(rules []store.Rule, branch string) branchPolicy {
	policy := branchPolicy{methods: slices.Clone(mergeMethods)}
	for _, rule := range rules {
		if matched, _ := path.Match(rule.Pattern, branch); !matched {
			continue
		}
		policy.rules = append(policy.rules, rule.Pattern)
		policy.requireSigned = policy.requireSigned || rule.RequireSigned
		policy.requireApproval = policy.requireApproval || rule.RequireApproval
		policy.requireChecks = policy.requireChecks || rule.RequireChecks
		policy.forbidForcePush = policy.forbidForcePush || !rule.AllowForcePush
		policy.forbidDeletion = policy.forbidDeletion || !rule.AllowDeletion
		policy.methods = slices.DeleteFunc(policy.methods, func(method string) bool { return !slices.Contains(rule.MergeMethods, method) })
	}
	return policy
}

func unverifiedCommits(keys []store.Key, commits []repos.Commit) ([]string, error) {
	signatures := []*repos.Signature{}
	for index := range commits {
		signatures = append(signatures, &commits[index].Signature)
	}
	if err := verifyWithKeys(keys, signatures...); err != nil {
		return nil, err
	}
	problems := []string{}
	for _, commit := range commits {
		if commit.Signature.Status != repos.Verified {
			problems = append(problems, fmt.Sprintf("commit %s (%s) is %s", commit.SHA[:7], commit.Subject, commit.Signature.Status))
		}
	}
	return problems, nil
}

func CheckPush(db *store.Store, git repos.Git, dir, repo string, updates []RefUpdate) error {
	rules, err := db.Rules(repo)
	if err != nil || len(rules) == 0 {
		return err
	}
	keys, err := db.Keys("")
	if err != nil {
		return err
	}
	for _, update := range updates {
		branch, isBranch := strings.CutPrefix(update.Ref, "refs/heads/")
		if !isBranch {
			continue
		}
		policy := policyFor(rules, branch)
		if zeroSHA(update.New) {
			if policy.forbidDeletion {
				return fmt.Errorf("branch rule %s protects %s from deletion", strings.Join(policy.rules, ", "), branch)
			}
			continue
		}
		if policy.forbidForcePush && !zeroSHA(update.Old) {
			forward, err := git.IsAncestor(dir, update.Old, update.New)
			if err != nil {
				return err
			}
			if !forward {
				return fmt.Errorf("branch rule %s refuses force pushes to %s; push commits on top of %s", strings.Join(policy.rules, ", "), branch, update.Old[:7])
			}
		}
		if !policy.requireSigned {
			continue
		}
		exclude := []string{}
		if !zeroSHA(update.Old) {
			exclude = append(exclude, update.Old)
		}
		commits, err := git.Log(dir, update.New, 10000, exclude...)
		if err != nil {
			return err
		}
		problems, err := unverifiedCommits(keys, commits)
		if err != nil {
			return err
		}
		if len(problems) > 0 {
			return fmt.Errorf("branch rule %s requires verified signatures on %s: %s", strings.Join(policy.rules, ", "), branch, strings.Join(problems, "; "))
		}
	}
	return nil
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	rules, err := s.store.Rules(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleSetRule(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	var body struct {
		Pattern         string   `json:"pattern"`
		RequireSigned   bool     `json:"require_signed"`
		RequireApproval bool     `json:"require_approval"`
		RequireChecks   bool     `json:"require_checks"`
		AllowForcePush  bool     `json:"allow_force_push"`
		AllowDeletion   bool     `json:"allow_deletion"`
		MergeMethods    []string `json:"merge_methods"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if _, err := path.Match(body.Pattern, ""); err != nil || body.Pattern == "" {
		writeError(w, http.StatusBadRequest, "pattern must be a branch name or glob, such as main or release/*")
		return
	}
	if len(body.MergeMethods) == 0 {
		body.MergeMethods = mergeMethods
	}
	for _, method := range body.MergeMethods {
		if !slices.Contains(mergeMethods, method) {
			writeError(w, http.StatusBadRequest, "merge methods must be ff or squash")
			return
		}
	}
	rule, eventID, err := s.store.SetRule(repo.Name, identityFrom(r).actor, store.Rule{
		Pattern: body.Pattern, RequireSigned: body.RequireSigned, RequireApproval: body.RequireApproval, RequireChecks: body.RequireChecks, AllowForcePush: body.AllowForcePush, AllowDeletion: body.AllowDeletion, MergeMethods: body.MergeMethods,
	})
	if err != nil {
		writeStoreError(w, err, "repository not found")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		store.Rule
		EventID int64 `json:"event_id"`
	}{rule, eventID})
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	pattern := r.URL.Query().Get("pattern")
	eventID, err := s.store.DeleteRule(repo.Name, identityFrom(r).actor, pattern)
	if err != nil {
		writeStoreError(w, err, "no rule for "+pattern)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": pattern, "event_id": eventID})
}
