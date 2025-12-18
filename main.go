package main

import (
	"fmt"
	"log"

	"github.com/mikluko/opentelemetry-collector-nats/internal/run"
)

var (
	version  = "undefined" // set at the build time
	revision = "undefined" // set at the build time
)

func main() {
	arg := fmt.Sprintf("%s (%s)", version, revision)
	if err := run.Main(arg); err != nil {
		log.Fatal(err)
	}
}
