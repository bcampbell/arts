package main

import (
	"fmt"
	"github.com/bcampbell/arts/arts"
	"gopkg.in/yaml.v2"
	"io"
)

// ugh. a bit of hoop-jumping so we can use the yaml encoder
// to write out the article front-matter section

type frontmatterArt struct {
	CanonicalURL string `yaml:"canonical_url,omitempty"`
	// all known URLs for article (including canonical)
	// TODO: first url should be considered "preferred" if no canonical?
	URLs     []string            `yaml:"urls,omitempty"`
	Headline string              `yaml:"headline,omitempty"`
	Authors  []frontmatterAuthor `yaml:"authors,omitempty"`
	//	Content  string   `json:"content,omitempty"`
	Published   string                 `yaml:"published,omitempty"`
	Updated     string                 `yaml:"updated,omitempty"`
	Publication frontmatterPublication `yaml:"publication,omitempty"`
	Keywords    []frontmatterKeyword   `yaml:"keywords,omitempty"`
	Section     string                 `yaml:"section,omitempty"`
	// TODO:
	// Language
	// article confidence?
}

type frontmatterAuthor struct {
	Name    string `yaml:"name"`
	RelLink string `yaml:"rellink,omitempty"`
	Email   string `yaml:"email,omitempty"`
	Twitter string `yaml:"twitter,omitempty"`
}

type frontmatterKeyword struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

type frontmatterPublication struct {
	Name   string `yaml:"name,omitempty"`
	Domain string `yaml:"domain,omitempty"`
}

func dumpArt(w io.Writer, art *arts.Article) error {

	// yaml front matter
	fmt.Fprintf(w, "---\n")

	pub2 := frontmatterPublication{
		Name:   art.Publication.Name,
		Domain: art.Publication.Domain,
	}

	authors2 := make([]frontmatterAuthor, len(art.Authors))
	for i, author := range art.Authors {
		authors2[i] = frontmatterAuthor{
			Name:    author.Name,
			RelLink: author.RelLink,
			Email:   author.Email,
			Twitter: author.Twitter,
		}
	}
	kwds2 := make([]frontmatterKeyword, len(art.Keywords))
	for i, kw := range art.Keywords {
		kwds2[i] = frontmatterKeyword{
			Name: kw.Name,
			URL:  kw.URL,
		}
	}

	art2 := frontmatterArt{
		CanonicalURL: art.CanonicalURL,
		URLs:         art.URLs,
		Headline:     art.Headline,
		Authors:      authors2,
		Published:    art.Published,
		Updated:      art.Updated,
		Publication:  pub2,
		Keywords:     kwds2,
		Section:      art.Section,
	}

	out, err := yaml.Marshal(art2)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "---\n")
	// the text content
	fmt.Fprint(w, art.Content)
	return nil
}
