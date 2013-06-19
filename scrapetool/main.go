package main

// commandline tool to grab, scrape and output a news article
// NOTE: currently assumes you've got a local http proxy running,
// to cache articles.
// I use squid, tweaked to cache for an hour or two, even if the web site
// says not to (which is really common. A lot of newspapers think the little
// clock in their page header is vitally important ;-)
// The idea is that the cachine proxy will be used by both article scraping,
// and article discovery (and maybe for other operations too). So if you need
// to crawl a site to find article, you won't need to hit the server again if
// the articles were part of the crawl.
//
// for now, I'm using this in my squid.conf:
//   refresh_pattern ^http: 60 20% 4320 ignore-no-cache ignore-no-store override-expire
//
//
import (
	"errors"
	"flag"
	"fmt"
	"github.com/bcampbell/arts"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// quote a string for yaml output
func quote(s string) string {
	if strings.ContainsAny(s, `:|`) {
		if !strings.Contains(s, `"`) {
			return fmt.Sprintf(`"%s"`, s)
		} else {
			if strings.Contains(s, "'") {
				s = strings.Replace(s, "'", "''", -1)
			}
			return fmt.Sprintf(`'%s'`, s)
		}
	}
	return s
}

func main() {
	var debug string
	flag.StringVar(&debug, "d", "", "log debug info to stderr (h=headline, c=content, a=authors d=dates all=hcad)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Usage: ", os.Args[0], "<article url>")
		os.Exit(1)
	}

	artURL := flag.Arg(0)
	u, err := url.Parse(artURL)
	if err != nil {
		panic(err)
	}

	// set up the debug logging
	debug = strings.ToLower(debug)
	if debug == "name" {
		debug = ""
	}
	if debug == "all" {
		debug = "hcad"
	}
	for _, flag := range debug {
		switch flag {
		case 'h':
			arts.Debug.HeadlineLogger = log.New(os.Stderr, "", 0)
		case 'c':
			arts.Debug.ContentLogger = log.New(os.Stderr, "", 0)
		case 'a':
			arts.Debug.AuthorsLogger = log.New(os.Stderr, "", 0)
		case 'd':
			arts.Debug.DatesLogger = log.New(os.Stderr, "", 0)
		}
	}

	var in io.ReadCloser
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		in, err = openHttp(artURL)
		if err != nil {
			panic(err)
		}
	case "file", "":
		in, err = os.Open(u.Path)
		if err != nil {
			panic(err)
		}
	}

	defer in.Close()
	raw_html, err := ioutil.ReadAll(in)
	if err != nil {
		panic(err)
	}

	art, err := arts.Extract(raw_html, artURL)
	if err != nil {
		panic(err)
	}

	writeYaml(os.Stdout, artURL, art)
}

func openHttp(artURL string) (io.ReadCloser, error) {
	proxyString := "http://localhost:3128"
	proxyURL, err := url.Parse(proxyString)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: transport}

	request, err := http.NewRequest("GET", artURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Request failed: %s", response.Status))
	}
	return response.Body, nil
}

// The plan is to store a big set of example articles in this format:
// YAML front matter (like in jekyll), with headline, authors etc...
// The rest of the file has the expected article text.
func writeYaml(w io.Writer, url string, art *arts.Article) {
	// yaml front matter
	fmt.Fprintf(w, "---\n")
	fmt.Fprintf(w, "canonical_url: %s\n", quote(art.CanonicalUrl))
	if len(art.AlternateUrls) > 0 {
		fmt.Fprintf(w, "alternate_urls:\n")
		for _, url := range art.AlternateUrls {
			fmt.Fprintf(w, "  - %s\n", quote(url))
		}
	}
	fmt.Fprintf(w, "headline: %s\n", quote(art.Headline))
	if len(art.Authors) > 0 {
		fmt.Fprintf(w, "authors:\n")
		for _, author := range art.Authors {
			fmt.Fprintf(w, "  - name: %s\n", quote(author.Name))
		}
	}
	if art.Published != "" {
		fmt.Fprintf(w, "published: %s\n", art.Published)
	}
	if art.Updated != "" {
		fmt.Fprintf(w, "updated: %s\n", art.Updated)
	}
	fmt.Fprintf(w, "---\n")
	// the text content
	fmt.Fprint(w, art.Content)
}
