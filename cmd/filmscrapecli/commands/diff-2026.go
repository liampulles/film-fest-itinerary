package commands

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// errTooManyRequests is returned by fetchDocument when the server responds with
// HTTP 429, indicating we are being rate limited.
var errTooManyRequests = errors.New("rate limited (HTTP 429 Too Many Requests)")

// mainURL is the DIFF 2026 film listing page.
const mainURL = "https://ccadiff.ukzn.ac.za/films-2026/"

// filmURLRe matches a DIFF film detail page, e.g.
// https://ccadiff.ukzn.ac.za/diff-47/boy-no-fear/
var filmURLRe = regexp.MustCompile(`^https?://[^/]+/diff-\d+/[a-z0-9-]+/?$`)

var (
	// filmYearRe matches a standalone 4-digit production year in the metadata row.
	filmYearRe = regexp.MustCompile(`^\d{4}$`)
	// filmDurationRe extracts the minutes from a value like "15Min".
	filmDurationRe = regexp.MustCompile(`(?i)(\d+)\s*min`)
	// filmScreeningRe matches a screening line, e.g. "1 Aug 12:00 Suncoast 5".
	filmScreeningRe = regexp.MustCompile(`^(\d{1,2}\s+[A-Za-z]{3})\s+(\d{1,2}:\d{2})\s+(.+)$`)
)

// filmTypePreference lists the film types in the order we prefer when a page
// declares more than one.
var filmTypePreference = []string{
	"DIFFXWOW Shorts",
	"Documentary",
	"Retrospective",
	"Feature",
	"Short Film",
	"Student Film",
}

// filmDetails describes a single film as scraped from its detail page.
type filmDetails struct {
	Name       string
	Year       int
	Duration   int // minutes
	Countries  string
	Languages  string
	Type       string
	Synopsis   string
	Screenings []screening
}

// screening is a single scheduled showing of a film.
type screening struct {
	Date   string // e.g. "1 Aug"
	Time   string // e.g. "12:00"
	Cinema string // e.g. "Suncoast 5"
}

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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errTooManyRequests
	}
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

// getFilmDetails fetches a single film detail page and extracts its details.
func getFilmDetails(pageURL string) (filmDetails, error) {
	doc, err := fetchDocument(pageURL)
	if err != nil {
		return filmDetails{}, err
	}

	single := doc.Find(".elementor-location-single").First()
	hero := single.Find("section.elementor-top-section").First()

	details := filmDetails{
		Name:       strings.TrimSpace(hero.Find("h2.elementor-heading-title").First().Text()),
		Type:       extractFilmType(hero),
		Synopsis:   extractSynopsis(single),
		Screenings: extractScreenings(single),
	}

	details.Year, details.Countries, details.Duration, details.Languages = parseMetadataRow(hero)

	return details, nil
}

// extractFilmType reads the type tag(s) above the title and returns the most
// preferred one per filmTypePreference. Falls back to the first declared type
// if none are listed.
func extractFilmType(hero *goquery.Selection) string {
	var types []string
	hero.Find("h5.elementor-heading-title a").Each(func(i int, s *goquery.Selection) {
		if t := strings.TrimSpace(s.Text()); t != "" {
			types = append(types, t)
		}
	})

	for _, pref := range filmTypePreference {
		for _, t := range types {
			if t == pref {
				return pref
			}
		}
	}
	if len(types) > 0 {
		return types[0]
	}
	return ""
}

// parseMetadataRow reads the "director | country | year | duration | language"
// row beneath the title. It anchors on the year (a standalone 4-digit value)
// and duration ("NNMin") rather than fixed positions; countries is the value
// immediately before the year and languages is the final value.
func parseMetadataRow(hero *goquery.Selection) (year int, countries string, duration int, languages string) {
	var values []string
	hero.Find(".elementor-widget-text-editor").Each(func(i int, s *goquery.Selection) {
		if v := cleanMetadataValue(s.Text()); v != "" {
			values = append(values, v)
		}
	})

	yearIdx := -1
	for i, v := range values {
		if filmYearRe.MatchString(v) {
			year, _ = strconv.Atoi(v)
			yearIdx = i
			break
		}
	}
	if yearIdx > 0 {
		countries = values[yearIdx-1]
	}

	for _, v := range values {
		if m := filmDurationRe.FindStringSubmatch(v); m != nil {
			duration, _ = strconv.Atoi(m[1])
			break
		}
	}

	if len(values) > 0 {
		languages = values[len(values)-1]
	}

	return year, countries, duration, languages
}

// cleanMetadataValue trims whitespace and the trailing " |" separator that the
// metadata widgets carry.
func cleanMetadataValue(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, " |")
	return strings.TrimSpace(s)
}

// extractSynopsis returns the text of the "Synopsis" accordion item.
func extractSynopsis(single *goquery.Selection) string {
	var synopsis string
	single.Find(".elementor-accordion-item").EachWithBreak(func(i int, s *goquery.Selection) bool {
		title := strings.TrimSpace(s.Find(".elementor-accordion-title").Text())
		if strings.EqualFold(title, "Synopsis") {
			synopsis = strings.TrimSpace(s.Find(".elementor-tab-content").Text())
			return false
		}
		return true
	})
	return synopsis
}

// extractScreenings finds every "D Mon HH:MM Cinema" line on the page.
func extractScreenings(single *goquery.Selection) []screening {
	var screenings []screening
	single.Find(".elementor-widget-text-editor").Each(func(i int, s *goquery.Selection) {
		text := strings.Join(strings.Fields(s.Text()), " ")
		m := filmScreeningRe.FindStringSubmatch(text)
		if m == nil {
			return
		}
		screenings = append(screenings, screening{
			Date:   m[1],
			Time:   m[2],
			Cinema: strings.TrimSpace(m[3]),
		})
	})
	return screenings
}
