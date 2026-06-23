package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	// scrapeTimeout bounds the whole detail-scraping run.
	scrapeTimeout = 15 * time.Minute
	// scrapeDelay is the initial delay between page fetches. The site rate
	// limits per-IP, so we fetch serially; the delay doubles on failure and
	// resets on success (see mapWithBackoff).
	scrapeDelay = 3 * time.Second
)

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

	urls, err := getFilmURLs(context.Background(), mainURL)
	if err != nil {
		return err
	}

	details, err := getAllFilmDetails(urls)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return nil
}

// fetchDocument retrieves url over HTTP and parses the response body into a
// goquery document. The request is bound to ctx, so a cancelled or expired
// context aborts an in-flight fetch.
func fetchDocument(ctx context.Context, url string) (*goquery.Document, error) {
	log.Printf("fetching %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("non-ok response for %s: %s", url, resp.Status)
		return nil, fmt.Errorf("http error: %s", resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// getFilmURLs fetches the main festival page and returns the URLs of every
// film detail page linked from it, de-duplicated and in first-seen order.
func getFilmURLs(ctx context.Context, pageURL string) ([]string, error) {
	doc, err := fetchDocument(ctx, pageURL)
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
func getFilmDetails(ctx context.Context, pageURL string) (filmDetails, error) {
	doc, err := fetchDocument(ctx, pageURL)
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

// getAllFilmDetails fetches the detail pages for every URL serially and returns
// the parsed details in the same order. The whole run is bounded by
// scrapeTimeout.
func getAllFilmDetails(urls []string) ([]filmDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	return mapWithBackoff(ctx, urls, getFilmDetails, scrapeDelay)
}

// mapWithBackoff applies fn to each input serially, in order, sleeping delay
// before every call. When a call fails (for any reason) it doubles the delay
// and retries the same input, continuing to double on each further failure;
// a success resets the delay to its initial value before moving on. The only
// thing that stops retrying is ctx — once it is cancelled or its deadline
// passes, mapWithBackoff returns ctx's error. Results are returned in input
// order.
func mapWithBackoff[In, Out any](
	ctx context.Context,
	inputs []In,
	fn func(context.Context, In) (Out, error),
	delay time.Duration,
) ([]Out, error) {
	results := make([]Out, 0, len(inputs))
	currentDelay := delay

	for _, in := range inputs {
		for {
			if err := sleepWithContext(ctx, currentDelay); err != nil {
				return nil, err
			}

			out, err := fn(ctx, in)
			if err == nil {
				results = append(results, out)
				currentDelay = delay
				break
			}

			// Failed — back off and retry the same input.
			currentDelay *= 2
		}
	}

	return results, nil
}

// sleepWithContext waits for d, returning early with ctx's error if it is
// cancelled first.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
		if slices.Contains(types, pref) {
			return pref
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
