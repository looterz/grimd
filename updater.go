package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var timesSeen = make(map[string]int)

// Update downloads all of the blocklists and imports them into the database
func Update() error {
	if _, err := os.Stat("lists"); os.IsNotExist(err) {
		if err := os.Mkdir("lists", 0600); err != nil {
			return fmt.Errorf("error creating lists directory: %s", err)
		}
	}

	if err := fetchSources(); err != nil {
		return fmt.Errorf("error fetching sources: %s", err)
	}

	return nil
}

func downloadFile(uri string, name string) error {
	filePath := filepath.FromSlash(fmt.Sprintf("lists/%s", name))

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
	files, err := ioutil.ReadDir("lists")
	if err != nil {
		return fmt.Errorf("could not read directory: %s", err)
	}

	for _, f := range files {
		file, err := os.Open(filepath.FromSlash(fmt.Sprintf("lists/%s", f.Name())))
		if err != nil {
			return fmt.Errorf("error opening file: %s", err)
		}
		defer file.Close()

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

				// TODO: translate to map[string]bool for better performance
				whitelisted := false
				for _, entry := range Config.Whitelist {
					if entry == line {
						whitelisted = true
					}
				}

				if !BlockCache.Exists(line) && !whitelisted {
					BlockCache.Set(line, true)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error scanning file: %s", err)
		}
	}

	for _, entry := range Config.Blocklist {
		BlockCache.Set(entry, true)
	}

	log.Printf("%d domains loaded from sources\n", BlockCache.Length())

	return nil
}
