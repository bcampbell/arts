package main

// commandline tool to grab, scrape and output a news article
//
// can grab article via http or from a file (raw html or the
// first response in a .warc)
//

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/bcampbell/arts/arts"
	"github.com/bcampbell/warc/warc"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/pprof"
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
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Usage: ", os.Args[0], "<article url>")
		os.Exit(1)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
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

	// TODO: sort all this mess out!
	artURL := flag.Arg(0)
	u, err := url.Parse(artURL)
	if err != nil {
		panic(err)
	}

	var in io.ReadCloser
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		in, err = openHttp(artURL)
		if err != nil {
			panic(err)
		}
	case "file", "":

		foo := strings.ToLower(u.Path)
		if strings.HasSuffix(foo, ".warc") {
			art, err := fromWARC(u.Path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			writeYaml(os.Stdout, art)
			return
		}

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

	art, err := arts.ExtractHTML(raw_html, artURL)
	if err != nil {
		panic(err)
	}

	writeYaml(os.Stdout, art)
}

func fromWARC(filename string) (*arts.Article, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	warcReader := warc.NewReader(in)
	for {
		//	fmt.Printf("WARC\n")
		rec, err := warcReader.ReadRecord()
		if err != nil {
			return nil, fmt.Errorf("Error reading %s: %s", filename, err)
		}
		if rec.Header.Get("Warc-Type") != "response" {
			continue
		}
		reqURL := rec.Header.Get("Warc-Target-Uri")
		// parse response, grab raw html
		rdr := bufio.NewReader(bytes.NewReader(rec.Block))
		response, err := http.ReadResponse(rdr, nil)
		if err != nil {
			return nil, fmt.Errorf("Error parsing response: %s", err)
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP error: %d", response.StatusCode)
		}
		rawHTML, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		// TODO: arts should allow parsing in raw response? or header + body?
		return arts.ExtractHTML(rawHTML, reqURL)
	}

}

func openHttp(artURL string) (io.ReadCloser, error) {

	client := &http.Client{}

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
// TODO: use a proper YAML encoder, dammit!
func writeYaml(w io.Writer, art *arts.Article) {
	// yaml front matter
	fmt.Fprintf(w, "---\n")
	fmt.Fprintf(w, "urls:\n")
	for _, url := range art.URLs {
		fmt.Fprintf(w, "  - %s\n", quote(url))
	}
	fmt.Fprintf(w, "canonical_url: %s\n", quote(art.CanonicalURL))
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
	fmt.Fprintf(w, "publication:\n")
	fmt.Fprintf(w, "  name: %s\n", quote(art.Publication.Name))
	fmt.Fprintf(w, "  domain: %s\n", quote(art.Publication.Domain))
	fmt.Fprintf(w, "---\n")
	// the text content
	fmt.Fprint(w, art.Content)
}
