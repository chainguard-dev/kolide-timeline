// Generate a CSV timeline file from Kolide events
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"k8s.io/klog/v2"
)

type OutFile struct {
	DiffResults       diffResults       `json:"diffResults"`
	Name              string            `json:"name"`
	Decorations       map[string]string `json:"decorations"`
	KolideDecorations KolideDecorations `json:"kolide_decorations"`
	UNIXTime          int64             `json:"unixTime"`
}

type KolideDecorations struct {
	DeviceOwnerEmail  string `json:"device_owner_email"`
	DeviceDisplayName string `json:"device_display_name"`
	DeviceOwnerType   string `json:"device_owner_type"`
}

type diffResults struct {
	Removed []Row
	Added   []Row
}

type Row map[string]string

func rowString(r Row) string {
	var sb strings.Builder
	for k, v := range r {
		if v == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s ", k, strings.TrimSpace(v)))
	}
	return strings.TrimSpace(sb.String())
}

func readFile(path string) (*OutFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Inconsistency warning: we've seen records returned as an array and as a struct
	out := &OutFile{}
	err = json.Unmarshal(body, out)

	// Try again by decoding it as an array
	if err != nil {
		outArr := []*OutFile{}
		errArr := json.Unmarshal(body, &outArr)
		if errArr != nil {
			klog.Errorf("unmarshal(%s): %v\nsecond attempt: %v", body, err, errArr)
			return nil, err
		}
		out = outArr[0]
	}

	return out, nil
}

type Event struct {
	Timestamp int64
	UTC       string
	Name      string
	Relation  string
	Line      string
}

func main() {
	path := os.Args[1]
	ofs := []*OutFile{}
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(info.Name(), ".json") {
				return nil
			}

			//	fmt.Println(path, info.Size())
			of, err := readFile(path)
			if err != nil {
				klog.Fatalf("read failed: %v", err)
			}
			if len(of.DiffResults.Added) > 0 {
				ofs = append(ofs, of)
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	// create events from inputs
	evs := []*Event{}

	polled := map[string]bool{}
	for _, o := range ofs {
		name := strings.ReplaceAll(o.Name, "pack:kolide_log_pipeline:", "")
		for _, r := range o.DiffResults.Added {
			line := rowString(r)
			for k, v := range r {
				if i, err := strconv.ParseInt(v, 10, 64); err == nil {
					if i > 1620836044 && i < time.Now().Unix() {
						evs = append(evs, &Event{
							Timestamp: i,
							Name:      name,
							Relation:  k,
							Line:      line,
						})
					}
				}
			}
			if polled[line] {
				klog.Infof("skipping dupe line: %v", line)
				continue
			}

			// poll logs don't make sense for events
			if strings.Contains(name, "events") {
				continue
			}

			evs = append(evs, &Event{
				Timestamp: o.UNIXTime,
				Name:      name,
				Relation:  "poll",
				Line:      line,
			})
			polled[line] = true
		}
	}

	sort.Slice(evs, func(i, j int) bool {
		return evs[i].Timestamp < evs[j].Timestamp
	})

	for _, e := range evs {
		e.UTC = time.Unix(e.Timestamp, 0).Format(time.UnixDate)
	}

	if err = gocsv.MarshalFile(&evs, os.Stdout); err != nil {
		klog.Fatalf("csv: %v", err)
	}
}
