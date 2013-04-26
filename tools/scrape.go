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
	"net/http"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"io"
	"strings"
	"arts"
	"flag"
)


// quote a string for yaml output
func quote(s string) string {
	if strings.Contains(s,`:`) {
		if !strings.Contains(s,`"`) {
			return fmt.Sprintf(`"%s"`,s)
		} else {
			if strings.Contains(s,"'") {
				s = strings.Replace(s, "'", "''", -1)
			}
			return fmt.Sprintf(`'%s'`,s)
		}
	}
	return s
}


func main() {
	var debug = flag.Bool("d", false, "log debug info to stderr")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Usage: ", os.Args[0], "<article url>")
		os.Exit(1)
	}


	proxyString := "http://localhost:3128"
	proxyURL, err := url.Parse(proxyString)
	if err != nil {
		panic(err)
	}
	artURL := flag.Arg(0)
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
	art,err := arts.Extract(raw_html,artURL,*debug)
	if err != nil {
		panic(err)
	}

	writeYaml(os.Stdout, artURL, art)
}


// The plan is to store a big set of example articles in this format:
// YAML front matter (like in jekyll), with headline, authors etc...
// The rest of the file has the expected article text.
func writeYaml(w io.Writer, url string, art *arts.Article) {
	// yaml front matter
	fmt.Fprintf(w,"---\n")
	fmt.Fprintf(w,"url: %s\n", quote(art.CanonicalUrl))
	fmt.Fprintf(w,"headline: %s\n", quote(art.Headline))
	if len(art.Authors)>0 {
		fmt.Fprintf(w,"authors:\n")
		for _,author := range art.Authors {
			fmt.Fprintf(w,"  - name: %s\n", quote(author.Name))
		}
	}
	fmt.Fprintf(w,"---\n")
	// the text content
	fmt.Fprint(w,art.Content)
}

