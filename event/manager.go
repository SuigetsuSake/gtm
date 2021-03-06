package event

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/git-time-metric/gtm/epoch"
	"github.com/git-time-metric/gtm/project"
)

// Record creates an event for a source
func Record(file string) error {
	sourcePath, gtmPath, err := pathFromSource(file)
	if err != nil {
		return err
	}

	if err := writeEventFile(sourcePath, gtmPath); err != nil {
		return err
	}

	return nil
}

// Process processes event files for a git repo
func Process(rootPath, gtmPath string, interim bool) (map[int64]map[string]int, error) {
	events := make(map[int64]map[string]int, 0)

	files, err := ioutil.ReadDir(gtmPath)
	if err != nil {
		return events, err
	}

	filesToRemove := []string{}
	var prevEpoch int64
	var prevFilePath string
	for i := range files {

		if !strings.HasSuffix(files[i].Name(), ".event") {
			continue
		}

		eventFilePath := filepath.Join(gtmPath, files[i].Name())
		filesToRemove = append(filesToRemove, eventFilePath)

		s := strings.SplitN(files[i].Name(), ".", 2)
		if len(s) != 2 {
			continue
		}

		fileEpoch, err := strconv.ParseInt(s[0], 10, 64)
		if err != nil {
			continue
		}
		fileEpoch = epoch.Minute(fileEpoch)

		sourcePath, err := readEventFile(eventFilePath)
		if err != nil {
			project.Log(fmt.Sprintf("\nRemoving corrupt event file %s, %s\n", eventFilePath, err))
			if err := os.Remove(eventFilePath); err != nil {
				project.Log(fmt.Sprintf("\nError removing event file %s, %s\n", eventFilePath, err))
			}
			continue
		}

		if _, ok := events[fileEpoch]; !ok {
			events[fileEpoch] = make(map[string]int, 0)
		}
		events[fileEpoch][sourcePath]++

		// Add idle events
		if prevEpoch != 0 && prevFilePath != "" {
			for e := prevEpoch + epoch.WindowSize; e < fileEpoch && e <= prevEpoch+epoch.IdleTimeout; e += epoch.WindowSize {
				if _, ok := events[e]; !ok {
					events[e] = make(map[string]int, 0)
				}
				events[e][prevFilePath]++
			}
		}
		prevEpoch = fileEpoch
		prevFilePath = sourcePath
	}

	if !interim {
		if err := removeFiles(filesToRemove); err != nil {
			return events, err
		}
	}

	return events, nil
}
