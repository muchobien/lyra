package server

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type RefUpdate struct {
	Ref string `json:"ref"`
	Old string `json:"old"`
	New string `json:"new"`
}

var closingKeyword = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):?\s+#(\d+)\b`)

func zeroSHA(sha string) bool {
	return strings.Trim(sha, "0") == ""
}

func closeReferenced(db *store.Store, repo string, actor store.Actor, text, commit string) ([]string, error) {
	messages := []string{}
	for _, match := range closingKeyword.FindAllStringSubmatch(text, -1) {
		number, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		issue, err := db.Item("issue", repo, number)
		if err != nil || issue.State != "open" {
			continue
		}
		closed := "closed"
		if _, _, err := db.EditItem("issue", repo, number, actor, store.ItemEdit{State: &closed, Commit: commit}); err != nil {
			return messages, err
		}
		messages = append(messages, fmt.Sprintf("closed issue #%d", number))
	}
	return messages, nil
}

func ProcessPush(db *store.Store, git repos.Git, dir, repo string, actor store.Actor, updates []RefUpdate) ([]string, error) {
	messages := []string{}
	defaultBranch := git.DefaultBranch(dir)
	closeIssues := func(text, commit string) error {
		closed, err := closeReferenced(db, repo, actor, text, commit)
		messages = append(messages, closed...)
		return err
	}

	syncedHeads := []string{}
	open, err := db.Pulls(repo, "open")
	if err != nil {
		return messages, err
	}
	for _, update := range updates {
		branch, isBranch := strings.CutPrefix(update.Ref, "refs/heads/")
		if !isBranch || zeroSHA(update.New) {
			continue
		}
		for _, pull := range open {
			if pull.Head != branch || pull.HeadSHA == update.New {
				continue
			}
			baseSHA, err := git.MergeBase(dir, "refs/heads/"+pull.Base, update.New)
			if err != nil {
				baseSHA = pull.BaseSHA
			}
			if err := git.UpdateRef(dir, fmt.Sprintf("refs/pull/%d/head", pull.Number), update.New, ""); err != nil {
				return messages, err
			}
			if _, err := db.SyncPull(repo, pull.Number, actor, update.New, baseSHA); err != nil {
				return messages, err
			}
			messages = append(messages, fmt.Sprintf("updated pull request #%d", pull.Number))
			synced := pull
			synced.HeadSHA, synced.BaseSHA = update.New, baseSHA
			messages = append(messages, QueuePullRuns(db, git, dir, repo, actor, synced, "synchronize")...)
			syncedHeads = append(syncedHeads, update.New)
		}
	}

	if open, err = db.Pulls(repo, "open"); err != nil {
		return messages, err
	}
	for _, update := range updates {
		branch, isBranch := strings.CutPrefix(update.Ref, "refs/heads/")
		if !isBranch || zeroSHA(update.New) {
			continue
		}
		for _, pull := range open {
			if pull.Base != branch {
				continue
			}
			merged, err := git.IsAncestor(dir, pull.HeadSHA, update.New)
			if err != nil {
				return messages, err
			}
			if !merged {
				continue
			}
			baseSHA := pull.BaseSHA
			if !zeroSHA(update.Old) {
				if mergeBase, err := git.MergeBase(dir, update.Old, pull.HeadSHA); err == nil {
					baseSHA = mergeBase
				}
			}
			if _, err := db.MarkPullMerged(repo, pull.Number, actor, pull.HeadSHA, baseSHA, "push"); err != nil {
				return messages, err
			}
			messages = append(messages, fmt.Sprintf("merged pull request #%d", pull.Number))
			if branch == defaultBranch {
				if err := closeIssues(pull.Title+"\n"+pull.Body, pull.HeadSHA); err != nil {
					return messages, err
				}
			}
		}
	}

	for _, update := range updates {
		if update.Ref != "refs/heads/"+defaultBranch || zeroSHA(update.New) {
			continue
		}
		exclude := []string{}
		if !zeroSHA(update.Old) {
			exclude = append(exclude, update.Old)
		}
		commits, err := git.Log(dir, update.New, 1000, exclude...)
		if err != nil {
			return messages, err
		}
		for index := len(commits) - 1; index >= 0; index-- {
			if err := closeIssues(commits[index].Subject+"\n"+commits[index].Body, commits[index].SHA); err != nil {
				return messages, err
			}
		}
	}
	messages = append(messages, QueueRuns(db, git, dir, repo, actor, updates)...)
	for _, sha := range syncedHeads {
		if err := db.AnnounceChecks(repo, sha); err != nil {
			return messages, err
		}
	}
	return messages, nil
}
