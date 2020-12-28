package handler

import (
	"github.com/wenerme/agos/pkg/whoami"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	whoami.Handler(w, r)
}
