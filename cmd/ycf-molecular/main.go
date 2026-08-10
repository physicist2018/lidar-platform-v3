// Command ycf-molecular is the deployment entry point for a Yandex Cloud
// Function that computes the pure molecular (Rayleigh) backscatter signal.
//
// The actual logic lives in pkg/lidar/ycf.
//
// For deployment to Yandex Cloud Functions, the entry point is main.Handler
// (the platform provides its own runtime main). The local main() below serves
// the handler over HTTP so the command can also be run and tested locally
// with `go run ./cmd/ycf-molecular`.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/physcist2018/lidar-platform-v3/pkg/lidar/ycf"
)

// Handler is the Yandex Cloud Functions entry point (main.Handler).
func Handler(w http.ResponseWriter, r *http.Request) {
	ycf.Handler(w, r)
}

// main runs the handler as a local HTTP server for development and testing.
func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("ycf-molecular: listening on %s", addr)
	if err := http.ListenAndServe(addr, http.HandlerFunc(Handler)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
