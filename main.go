package main

import (
	"log"

	"github.com/mikluko/nats-otel-collector/internal/run"
)

var (
	version = "undefined" // set at the build time
)

func main() {
	if err := run.Main(version); err != nil {
		log.Fatal(err)
	}
}
