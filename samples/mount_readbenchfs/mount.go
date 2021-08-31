package main

import (
	"context"
	"flag"
	"log"
	"os"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/samples/readbenchfs"
)

var fMountPoint = flag.String("mount_point", "", "Path to mount point.")
var fReadOnly = flag.Bool("read_only", false, "Mount in read-only mode.")
var fVectored = flag.Bool("vectored", false, "Use vectored read.")
var fDebug = flag.Bool("debug", false, "Enable debug logging.")

func main() {
	flag.Parse()

	server, err := readbenchfs.NewReadBenchServer()
	if err != nil {
		log.Fatalf("makeFS: %v", err)
	}

	// Mount the file system.
	if *fMountPoint == "" {
		log.Fatalf("You must set --mount_point.")
	}

	cfg := &fuse.MountConfig{
		ReadOnly: *fReadOnly,
		UseVectoredRead: *fVectored,
	}

	if *fDebug {
		cfg.DebugLogger = log.New(os.Stderr, "fuse: ", 0)
	}

	mfs, err := fuse.Mount(*fMountPoint, server, cfg)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}

	// Wait for it to be unmounted.
	if err = mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join: %v", err)
	}
}
