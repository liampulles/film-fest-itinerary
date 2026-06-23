package commands

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// mainURL is the DIFF 2026 film listing page.
const mainURL = "https://ccadiff.ukzn.ac.za/films-2026/"

// filmURLRe matches a DIFF film detail page, e.g.
// https://ccadiff.ukzn.ac.za/diff-47/boy-no-fear/
var filmURLRe = regexp.MustCompile(`^https?://[^/]+/diff-\d+/[a-z0-9-]+/?$`)

func RunDIFF2026(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: filmscrapecli diff-2026")
	}

	urls, err := getFilmURLs(mainURL)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", urls)

	return nil
}

// fetchDocument retrieves url over HTTP and parses the response body into a
// goquery document.
func fetchDocument(url string) (*goquery.Document, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %s", resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// getFilmURLs fetches the main festival page and returns the URLs of every
// film detail page linked from it, de-duplicated and in first-seen order.
func getFilmURLs(pageURL string) ([]string, error) {
	doc, err := fetchDocument(pageURL)
	if err != nil {
		return nil, err
	}

	var urls []string
	seen := make(map[string]struct{})

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		if !filmURLRe.MatchString(href) {
			return
		}
		if _, dup := seen[href]; dup {
			return
		}
		seen[href] = struct{}{}
		urls = append(urls, href)
	})

	return urls, nil
}
