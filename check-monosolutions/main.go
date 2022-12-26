package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

const ConfDir = "some-path"

const (
	prefixMono = "mono."
	suffixConf = ".conf"
)

type SiteStatus struct {
	domain string
	ok     bool
	code   int
}

type SiteStatusReport struct {
	SiteStatus
	error
}

var reConf = regexp.MustCompile(suffixConf)
var reMono = regexp.MustCompile(prefixMono)

func main() {
	var parsedStatuses []SiteStatusReport

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	filesCheckChan := make(chan SiteStatusReport)
	files, err := os.ReadDir(homeDir + ConfDir)

	if err != nil {
		panic(err)
	}

	files = filter(files, func(el os.DirEntry) bool {
		return strings.Contains(el.Name(), prefixMono) && strings.Contains(el.Name(), suffixConf)
	})

	go func() {
		wg := sync.WaitGroup{}

		for _, file := range files {
			wg.Add(1)

			go func(entry os.DirEntry) {
				defer wg.Done()

				domain := reMono.ReplaceAllString(
					reConf.ReplaceAllString(entry.Name(), ""),
					"",
				)

				siteStatus, err := processSite(domain)

				filesCheckChan <- SiteStatusReport{siteStatus, err}
			}(file)
		}

		wg.Wait()
		close(filesCheckChan)
	}()

	for statusReport := range filesCheckChan {
		parsedStatuses = append(parsedStatuses, statusReport)
	}

	for _, status := range parsedStatuses {
		if status.error != nil {
			errorMessage := fmt.Sprintf("(%s)", status.error)
			println(status.domain, status.ok, status.code, errorMessage)
		}
	}

	println("ok: ", len(filter(parsedStatuses, func(status SiteStatusReport) bool {
		return status.ok
	})))
	println("err: ", len(filter(parsedStatuses, func(status SiteStatusReport) bool {
		return !status.ok
	})))
}

func processSite(domain string) (status SiteStatus, err error) {
	res, err := http.Get(fmt.Sprintf("https://%s", domain))

	if err != nil {
		return SiteStatus{
			domain: domain,
			ok:     false,
			code:   1,
		}, err
	}

	return SiteStatus{
		domain: domain,
		ok:     res.StatusCode >= 200 && res.StatusCode <= 303,
		code:   res.StatusCode,
	}, err
}

func filter[T any](dataset []T, test func(T) bool) (res []T) {
	for _, el := range dataset {
		if test(el) {
			res = append(res, el)
		}
	}
	return
}
