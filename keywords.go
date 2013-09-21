package arts

import (
	"code.google.com/p/go.net/html"
	"github.com/matrixik/goquery"
	"strings"
)

func grabKeywords(root *html.Node) (list []string) {
	doc := goquery.NewDocumentFromNode(root)
	keywordString, ok := doc.Find(`head meta[name="keywords"]`).Attr("content")
	if !ok || keywordString == "" {
		return
	}

	list = strings.Split(keywordString, ",")
	for n, k := range list {
		list[n] = strings.TrimSpace(k)
	}
	return
}
