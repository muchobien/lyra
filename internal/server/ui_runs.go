package server

import (
	"fmt"
	"net/http"
	"strconv"
)

func uiRunNumber(r *http.Request) (int64, error) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		return 0, notFound("Run not found.")
	}
	return number, nil
}

func (s *Server) uiRunAction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	id := identityFrom(r)
	if !id.can(scopeWrite, repo.Name) {
		return forbidden("This token can't change runs in " + repo.Name + ".")
	}
	number, err := uiRunNumber(r)
	if err != nil {
		return err
	}
	switch r.PathValue("action") {
	case "cancel":
		_, err = s.store.CancelRun(ctx, id.actor, repo.Name, number)
	case "rerun":
		_, err = s.store.RerunRun(ctx, id.actor, repo.Name, number, r.FormValue("failed") == "true")
	default:
		return notFound("Unknown action.")
	}
	if err != nil {
		return missing(err, fmt.Sprintf("Run #%d not found.", number))
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/runs/%d", repo.Name, number), http.StatusSeeOther)
	return nil
}
