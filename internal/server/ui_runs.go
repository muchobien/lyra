package server

import (
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
