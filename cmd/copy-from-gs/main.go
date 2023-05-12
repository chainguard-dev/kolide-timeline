// Copy files for a kolide device out of GCP
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"k8s.io/klog/v2"
)

var (
	bucketFlag         = flag.String("bucket", "", "Bucket to query")
	deviceIDFlag       = flag.Int("device-id", -1, "device ID to fetch logs for")
	prefixFlag         = flag.String("prefix", "", "directory of contents to query")
	excludeSubDirsFlag = flag.String("exclude-subdirs", "", "exclude alerts for this comma-separated list of subdirectories")
	maxAgeFlag         = flag.Duration("max-age", 10*time.Minute, "Maximum age of events to include (for best use, use at least 2X your trigger time)")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := context.Background()

	cutoff := time.Now().Add(*maxAgeFlag * -1)

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	bucketName := os.Getenv("BUCKET_NAME")
	if *bucketFlag != "" {
		bucketName = *bucketFlag
	}

	// Creates a Bucket instance.
	bucket := client.Bucket(bucketName)
	bucketPrefix := os.Getenv("BUCKET_PREFIX")
	if *prefixFlag != "" {
		bucketPrefix = *prefixFlag
	}

	excludeSubDirs := os.Getenv("EXCLUDE_SUBDIRS")
	if *excludeSubDirsFlag != "" {
		excludeSubDirs = *excludeSubDirsFlag
	}

	cc := &CollectConfig{Prefix: bucketPrefix, ExcludeSubdirs: strings.Split(excludeSubDirs, ","), Cutoff: cutoff, DeviceID: *deviceIDFlag}
	syncFiles(ctx, bucket, cc)
}

type CollectConfig struct {
	Prefix         string
	DeviceID       int
	Cutoff         time.Time
	ExcludeSubdirs []string
}

func syncFiles(ctx context.Context, bucket *storage.BucketHandle, cc *CollectConfig) []string {
	synced := []string{}
	klog.Infof("finding items matching: %+v ...", cc)
	it := bucket.Objects(ctx, &storage.Query{Prefix: cc.Prefix})
	maxEmptySize := int64(128)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			klog.Errorf("error fetching objects: %v", err)
			break
		}

		matched := false
		for _, d := range cc.ExcludeSubdirs {
			if strings.Contains(attrs.Name, "/"+d+"/") {
				matched = true
				break
			}
		}

		if matched || attrs.Created.Before(cc.Cutoff) {
			continue
		}

		if attrs.Size <= maxEmptySize {
			klog.V(1).Infof("skipping %s -- smaller than %d bytes", attrs.Name, attrs.Size)
			continue
		}

		if !strings.Contains(attrs.Name, fmt.Sprintf("device-%d", cc.DeviceID)) {
			klog.V(1).Infof("skipping %s - does not match device-%d", attrs.Name, cc.DeviceID)
			continue
		}

		if s, err := os.Stat(attrs.Name); err == nil {
			if s.Size() == attrs.Size {
				klog.Infof("skipping %s - already exists with %d bytes", attrs.Name, attrs.Size)
				continue
			}
			klog.Infof("found %s - but it contains %d bytes instead of %d (remote)", attrs.Name, attrs.Size, s.Size())
		}

		klog.Infof("reading: %+v (%d bytes)", attrs.Name, attrs.Size)
		rc, err := bucket.Object(attrs.Name).NewReader(ctx)
		if err != nil {
			klog.Fatal(err)
		}
		defer rc.Close()
		bs, err := io.ReadAll(rc)
		if err != nil {
			klog.Fatal(err)
		}

		klog.Infof("read %d bytes, writing to %s", len(bs), attrs.Name)
		if err := os.MkdirAll(filepath.Dir(attrs.Name), 0o755); err != nil {
			klog.Fatalf("unable to create directory: %s", err)
		}
		if err := os.WriteFile(attrs.Name, bs, 0o444); err != nil {
			klog.Fatalf("write failed: %v", err)
		}
		klog.Infof("saved %s (%d bytes)", attrs.Name, len(bs))

	}
	return synced
}
