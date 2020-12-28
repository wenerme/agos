package handler

import (
	"github.com/wenerme/agos/pkg/apis"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	apis.Router().ServeHTTP(w, r)
}
