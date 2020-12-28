package main

import (
	"github.com/wenerme/agos/pkg/apis"
	"net/http"
)

func main() {
	panic(http.ListenAndServe(":8081", apis.Router()))
}
