package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var timesSeen = make(map[string]int)
var whitelist = make(map[string]bool)

// Update downloads all of the blocklists and imports them into the database
func Update() error {
	if _, err := os.Stat("sources"); os.IsNotExist(err) {
		if err := os.Mkdir("sources", 0700); err != nil {
			return fmt.Errorf("error creating sources directory: %s", err)
		}
	}

	for _, entry := range Config.Whitelist {
		whitelist[entry] = true
	}

	for _, entry := range Config.Blocklist {
		BlockCache.Set(entry, true)
	}

	if err := fetchSources(); err != nil {
		return fmt.Errorf("error fetching sources: %s", err)
	}

	return nil
}

func downloadFile(uri string, name string) error {
	filePath := filepath.FromSlash(fmt.Sprintf("sources/%s", name))

	output, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %s", err)
	}
	defer output.Close()

	response, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("error downloading source: %s", err)
	}
	defer response.Body.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		return fmt.Errorf("error copying output: %s", err)
	}

	return nil
}

func fetchSources() error {
	var wg sync.WaitGroup

	for _, uri := range Config.Sources {
		wg.Add(1)

		u, _ := url.Parse(uri)
		host := u.Host
		timesSeen[host] = timesSeen[host] + 1
		fileName := fmt.Sprintf("%s.%d.list", host, timesSeen[host])

		go func(uri string, name string) {
			log.Printf("fetching source %s\n", uri)
			if err := downloadFile(uri, name); err != nil {
				fmt.Println(err)
			}

			wg.Done()
		}(uri, fileName)
	}

	wg.Wait()

	return nil
}

// UpdateBlockCache updates the BlockCache
func UpdateBlockCache() error {
	log.Printf("loading blocked domains from %d locations...\n", len(Config.SourceDirs))

	for _, dir := range Config.SourceDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Printf("directory %s not found, skipping\n", dir)
			continue
		}

		err := filepath.Walk(dir, func(path string, f os.FileInfo, _ error) error {
			if !f.IsDir() {
				file, err := os.Open(filepath.FromSlash(path))
				if err != nil {
					return fmt.Errorf("error opening file: %s", err)
				}
				defer file.Close()

				if err = parseHostFile(file); err != nil {
					return fmt.Errorf("error parsing hostfile %s", err)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("error walking location %s\n", err)
		}
	}

	log.Printf("%d domains loaded from sources\n", BlockCache.Length())

	return nil
}

func parseHostFile(file *os.File) error {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		isComment := strings.HasPrefix(line, "#")

		if !isComment && line != "" {
			fields := strings.Fields(line)

			if len(fields) > 1 && !strings.HasPrefix(fields[1], "#") {
				line = fields[1]
			} else {
				line = fields[0]
			}

			if !BlockCache.Exists(line) && !whitelist[line] {
				BlockCache.Set(line, true)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning hostfile: %s", err)
	}

	return nil
}
