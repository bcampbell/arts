package main

import (
	"net/http"
	//    "net/http/httputil"
	"code.google.com/p/go.net/html"
//	"code.google.com/p/go.net/html/atom"
	"fmt"
	"github.com/matrixik/goquery"
	"io/ioutil"
	"log"
	"net/url"
	"os"
//	"regexp"
//	"sort"
	"strings"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "<article url>")
		os.Exit(1)
	}
	proxyString := "http://localhost:3128"
	proxyURL, err := url.Parse(proxyString)
	if err != nil {
		panic(err)
	}
	artURL := os.Args[1]
	_, err = url.Parse(artURL)
	if err != nil {
		panic(err)
	}

	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: transport}

	request, err := http.NewRequest("GET", artURL, nil)
	if err != nil {
		panic(err)
	}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

    if response.StatusCode != 200 {
        fmt.Printf("Request failed: %s\n", response.Status)
        os.Exit(1)
    }

	foo, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	raw_html := string(foo)
	extract(raw_html,artURL)
}

func dbug(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func extract(raw_html, art_url string) {
	r := strings.NewReader(raw_html)
	root, err := html.Parse(r)
	if err != nil {
		panic(err)
	}

	doc := goquery.NewDocumentFromNode(root)
	extract_headline(doc,art_url)
	extractAuthor(doc)

	removeScripts(root)
	// TODO: Turn all double br's into p's? Kill <style> tags? (see prepDocument())
	contentNodes,contentScores := grabContent(root)
	removeCruft(contentNodes, contentScores)
	sanitiseContent(contentNodes)

	fmt.Printf("extracted %d nodes:\n", len(contentNodes))
	for _, n := range contentNodes {
		dumpTree(n, 0)
		//		fmt.Printf("%s:\n", describeNode(n))
		//		html.Render(os.Stdout, n)
	}
}


