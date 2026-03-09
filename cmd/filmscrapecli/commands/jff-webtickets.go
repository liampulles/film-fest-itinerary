package commands

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	durationRe   = regexp.MustCompile(`(?i)(\d+)\s*h(?:r)?\s*(\d+)?`)
	lineRe       = regexp.MustCompile(`(?i)^(.+)\s*\(([^()]*)\).*?(\d{2}\s+[a-z]{3}\s+\d{4}\s+\d{1,2}[:h\.]\d{2})\s*$`)
	doubleLineRe = regexp.MustCompile(`(?i)^(.+?)\s*\(([^)]+)\)\s*\+\s*(.+?)\s*\(([^)]+)\).*?(\d{2}\s+[a-z]{3}\s+\d{4}\s+\d{1,2}[:h\.]\d{2})\s*$`)
	dateTimeRe   = regexp.MustCompile(`(?i)\d{2}\s+[a-zA-Z]{3}\s+\d{4}\s+\d{1,2}\s*[:h\.]\s*\d{2}`)
	minutesRe    = regexp.MustCompile(`(?i)(\d+)\s*m(?:in)?`)
)

func RunJFFWebtickets(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: filmscrapecli jff-webtickets <cinema-name> <webtickets-url>")
	}

	cinemaName := strings.TrimSpace(args[0])
	url := args[1]

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation("Africa/Johannesburg")
	if err != nil {
		return err
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// CSV header
	writer.Write([]string{
		"cinema_name",
		"name",
		"start_datetime",
		"duration_minutes",
		"q_and_a",
	})

	seen := make(map[string]struct{})
	lines := extractCandidateLines(doc)

	for _, line := range lines {
		firstFilm, secondFilm, ok := parseDoubleFilmLine(line, loc)
		if ok {
			if !markSeen(seen, firstFilm) {
				writeFilmRow(writer, cinemaName, firstFilm)
			}

			if !markSeen(seen, secondFilm) {
				writeFilmRow(writer, cinemaName, secondFilm)
			}
			continue
		}

		film, ok := parseFilmLine(line, loc)
		if ok {
			if !markSeen(seen, film) {
				writeFilmRow(writer, cinemaName, film)
			}
			continue
		}
	}

	return nil
}

type filmLine struct {
	Name            string
	StartTime       time.Time
	DurationMinutes int
	QandA           bool
}

func parseFilmLine(line string, loc *time.Location) (filmLine, bool) {
	matches := lineRe.FindStringSubmatch(line)
	if matches == nil {
		return filmLine{}, false
	}

	name := strings.TrimSpace(matches[1])
	durationStr := strings.TrimSpace(matches[2])
	dateTimeStr := strings.TrimSpace(matches[3])

	if name == "" || dateTimeStr == "" || durationStr == "" {
		return filmLine{}, false
	}

	startTime, err := parseDateTime(dateTimeStr, loc)
	if err != nil {
		log.Printf("skipping %q: %v", name, err)
		return filmLine{}, false
	}

	durationMinutes, hasQA, err := parseDuration(durationStr)
	if err != nil {
		log.Printf("invalid duration %q for %q", durationStr, name)
		return filmLine{}, false
	}
	if hasQA {
		log.Printf("Q&A detected for %q (%s)", name, durationStr)
	}

	return filmLine{
		Name:            name,
		StartTime:       startTime,
		DurationMinutes: durationMinutes,
		QandA:           hasQA,
	}, true
}

func parseDoubleFilmLine(line string, loc *time.Location) (filmLine, filmLine, bool) {
	matches := doubleLineRe.FindStringSubmatch(line)
	if matches == nil {
		return filmLine{}, filmLine{}, false
	}

	firstName := strings.TrimSpace(matches[1])
	firstDurationStr := strings.TrimSpace(matches[2])
	secondName := strings.TrimSpace(matches[3])
	secondDurationStr := strings.TrimSpace(matches[4])
	dateTimeStr := strings.TrimSpace(matches[5])

	if firstName == "" || secondName == "" || firstDurationStr == "" || secondDurationStr == "" || dateTimeStr == "" {
		return filmLine{}, filmLine{}, false
	}

	startTime, err := parseDateTime(dateTimeStr, loc)
	if err != nil {
		log.Printf("skipping %q: %v", firstName, err)
		return filmLine{}, filmLine{}, false
	}

	firstDurationMinutes, hasQA, err := parseDuration(firstDurationStr)
	if err != nil {
		log.Printf("invalid duration %q for %q", firstDurationStr, firstName)
		return filmLine{}, filmLine{}, false
	}
	if hasQA {
		log.Printf("Q&A detected for %q (%s)", firstName, firstDurationStr)
	}

	secondDurationMinutes, hasQA, err := parseDuration(secondDurationStr)
	if err != nil {
		log.Printf("invalid duration %q for %q", secondDurationStr, secondName)
		return filmLine{}, filmLine{}, false
	}
	if hasQA {
		log.Printf("Q&A detected for %q (%s)", secondName, secondDurationStr)
	}

	secondStartTime := startTime.Add(time.Duration(firstDurationMinutes) * time.Minute)

	return filmLine{
			Name:            firstName,
			StartTime:       startTime,
			DurationMinutes: firstDurationMinutes,
		}, filmLine{
			Name:            secondName,
			StartTime:       secondStartTime,
			DurationMinutes: secondDurationMinutes,
		}, true
}

func writeFilmRow(writer *csv.Writer, cinemaName string, film filmLine) {
	writer.Write([]string{
		cinemaName,
		film.Name,
		film.StartTime.Format(time.RFC3339),
		strconv.Itoa(film.DurationMinutes),
		strconv.FormatBool(film.QandA),
	})
}

func markSeen(seen map[string]struct{}, film filmLine) bool {
	key := film.Name + "|" + film.StartTime.Format(time.RFC3339)
	if _, ok := seen[key]; ok {
		return true
	}
	seen[key] = struct{}{}
	return false
}

func extractCandidateLines(doc *goquery.Document) []string {
	lines := []string{}

	doc.Find("h4.text-body").Each(func(i int, s *goquery.Selection) {
		normalized := strings.Join(strings.Fields(s.Text()), " ")
		if normalized == "" {
			return
		}
		if !strings.Contains(normalized, "(") || !strings.Contains(normalized, ")") {
			return
		}
		if !dateTimeRe.MatchString(normalized) {
			return
		}
		lines = append(lines, normalized)
	})

	return lines
}

func parseDateTime(dateTimeStr string, loc *time.Location) (time.Time, error) {
	// Example: "04 Mar 2026 15:30"
	layout := "02 Jan 2006 15:04"
	normalized := strings.Join(strings.Fields(dateTimeStr), " ")
	return time.ParseInLocation(layout, normalized, loc)
}

func parseDuration(s string) (int, bool, error) {
	lower := strings.ToLower(s)
	hasQA := strings.Contains(lower, "q&a") || strings.Contains(lower, "q & a")
	cleaned := strings.ReplaceAll(lower, "q&a", "")
	cleaned = strings.ReplaceAll(cleaned, "q & a", "")
	cleaned = strings.ReplaceAll(cleaned, "+", " ")
	cleaned = strings.ReplaceAll(cleaned, "&", " ")
	cleaned = strings.ReplaceAll(cleaned, "’", "")
	cleaned = strings.ReplaceAll(cleaned, "'", "")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	m := durationRe.FindStringSubmatch(cleaned)
	if m != nil {
		hours, _ := strconv.Atoi(m[1])
		minutes := 0
		if len(m) > 2 && m[2] != "" {
			minutes, _ = strconv.Atoi(m[2])
		}
		return hours*60 + minutes, hasQA, nil
	}

	m = minutesRe.FindStringSubmatch(cleaned)
	if m != nil {
		minutes, _ := strconv.Atoi(m[1])
		return minutes, hasQA, nil
	}

	if cleaned != "" {
		if minutes, err := strconv.Atoi(cleaned); err == nil {
			return minutes, hasQA, nil
		}
	}

	return 0, hasQA, fmt.Errorf("unrecognized duration format")
}
