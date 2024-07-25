package main

import (
	"context"
	"flag"
	"fmt"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	log "github.com/sirupsen/logrus"
	"os"
	"xDS/internal/observer"
	"xDS/internal/processor"
	"xDS/internal/server"
)

var (
	l log.FieldLogger

	watchDirectory string
	port           uint
	nodeID         string
)

func init() {
	l = log.New()
	log.SetLevel(log.DebugLevel)

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18001, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "node", "default", "Node ID")

	// Define the directory to watch for Envoy configuration files
	flag.StringVar(&watchDirectory, "path", "config", "directory to watch for files")
}

func main() {
	flag.Parse()

	// Create a cache
	cache := cachev3.NewSnapshotCache(false, cachev3.IDHash{}, l)

	// Create a processor
	proc := processor.NewProcessor(cache, nodeID)

	// Create initial snapshot from file
	// Get all files in the watch directory
	entries, err := os.ReadDir(watchDirectory)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		msg := observer.NotifyMessage{
			Operation: observer.Create,
			FilePath:  fmt.Sprintf("%s/%s", watchDirectory, e.Name()),
		}
		proc.ProcessFile(msg)
	}

	// Notify channel for file system events
	notifyCh := make(chan observer.NotifyMessage)

	go func() {
		// Watch for file changes
		observer.Watch(watchDirectory, notifyCh)
	}()

	go func() {
		// Run the xDS server
		ctx := context.Background()
		srv := serverv3.NewServer(ctx, cache, nil)
		server.RunServer(srv, port)
	}()

	// Periodically process file
	for {
		select {
		case msg := <-notifyCh:
			proc.ProcessFile(msg)
		}
	}
}
