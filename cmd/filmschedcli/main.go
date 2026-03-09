package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/liampulles/film-fest-itinerary/pkg/interval"
)

// Example command line use:
//
//	filmfestschedcli < input.csv > output.csv
//
// Example input.csv
//
//	cinema_name,name,start_datetime,duration_minutes
//	artistry,CAR AIRCON ELECTRICALl AUTOCARTOONS,2026-03-04T15:00:00+02:00,89
//	artistry,DR. MBONGENI NGEMA,2026-03-04T17:00:00+02:00,60
//	bioscope,THE DUTCHMAN,2026-03-06T18:00:00+02:00,88
//	bioscope,HISTORY OF SOUND,2026-03-06T20:00:00+02:00,127
//	theatre-on-square,"TUKKI, FROM ROOTS TO BAYOU",2026-03-04T17:30:00+02:00,92
//	theatre-on-square,HISTORY OF SOUND,2026-03-04T20:00:00+02:00,127
//
// Example output.csv
//
//	schedule_number,cinema_name,name,start_datetime,duration_minutes
//	1,artistry,CAR AIRCON ELECTRICALl AUTOCARTOONS,2026-03-04T15:00:00+02:00,89
//	1,artistry,DR. MBONGENI NGEMA,2026-03-04T17:00:00+02:00,60
//	1,theatre-on-square,HISTORY OF SOUND,2026-03-04T20:00:00+02:00,127
//	1,bioscope,THE DUTCHMAN,2026-03-06T18:00:00+02:00,88
//	1,bioscope,HISTORY OF SOUND,2026-03-06T20:00:00+02:00,127
//	2,artistry,CAR AIRCON ELECTRICALl AUTOCARTOONS,2026-03-04T15:00:00+02:00,89
//	2,theatre-on-square,"TUKKI, FROM ROOTS TO BAYOU",2026-03-04T17:30:00+02:00,92
//	2,theatre-on-square,HISTORY OF SOUND,2026-03-04T20:00:00+02:00,127
//	2,bioscope,THE DUTCHMAN,2026-03-06T18:00:00+02:00,88
//	2,bioscope,HISTORY OF SOUND,2026-03-06T20:00:00+02:00,127
func main() {
	// Read groups
	groups, cinemaByShowing, err := readGroups()
	if err != nil {
		log.Fatal(err)
	}

	// Create schedules
	schedules := interval.GroupIntervalSchedulingMaximization(groups)

	// Output schedules
	if err := writeSchedulesCSV(schedules, cinemaByShowing); err != nil {
		log.Fatal(err)
	}
}

type showingKey struct {
	FilmName  string
	StartUnix int64
}

func readGroups() ([]interval.Group[string, int64], map[showingKey]string, error) {
	reader := csv.NewReader(os.Stdin)
	reader.FieldsPerRecord = 4

	byName := make(map[string][]interval.Interval[int64])
	cinemaByShowing := make(map[showingKey]string)
	line := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read csv: %w", err)
		}

		line++
		if line == 1 && isHeader(record) {
			continue
		}

		cinemaName := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])
		startStr := strings.TrimSpace(record[2])
		durationStr := strings.TrimSpace(record[3])
		if cinemaName == "" || name == "" || startStr == "" || durationStr == "" {
			return nil, nil, fmt.Errorf("line %d: missing cinema name, film name, start time, or duration", line)
		}

		startTime, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: parse start time: %w", line, err)
		}

		durationMinutes, err := strconv.Atoi(durationStr)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: parse duration minutes: %w", line, err)
		}

		endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)
		iv, err := interval.New(startTime.Unix(), endTime.Unix())
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: build interval: %w", line, err)
		}

		byName[name] = append(byName[name], iv)
		cinemaByShowing[showingKey{FilmName: name, StartUnix: startTime.Unix()}] = cinemaName
	}

	var groups []interval.Group[string, int64]
	for name, intervals := range byName {
		groups = append(groups, interval.Group[string, int64]{
			Key:       name,
			Intervals: intervals,
		})
	}

	return groups, cinemaByShowing, nil
}

func isHeader(record []string) bool {
	return strings.EqualFold(strings.TrimSpace(record[0]), "cinema_name")
}

func writeSchedulesCSV(
	schedules []interval.Schedule[string, int64],
	cinemaByShowing map[showingKey]string,
) error {
	loc, err := time.LoadLocation("Africa/Johannesburg")
	if err != nil {
		return fmt.Errorf("load location: %w", err)
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	if err := writer.Write([]string{
		"schedule_number",
		"cinema_name",
		"name",
		"start_datetime",
		"duration_minutes",
	}); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, schedule := range schedules {
		scheduleNumber := i + 1
		for _, filmName := range schedule.OrderedKeys() {
			iv := schedule[filmName]
			start := time.Unix(iv.Start(), 0).In(loc)
			end := time.Unix(iv.End(), 0).In(loc)
			durationMinutes := int(end.Sub(start).Minutes())
			cinemaName, ok := cinemaByShowing[showingKey{FilmName: filmName, StartUnix: iv.Start()}]
			if !ok {
				return fmt.Errorf("cinema not found for %q at %s", filmName, start.Format(time.RFC3339))
			}
			if err := writer.Write([]string{
				strconv.Itoa(scheduleNumber),
				cinemaName,
				filmName,
				start.Format(time.RFC3339),
				strconv.Itoa(durationMinutes),
			}); err != nil {
				return fmt.Errorf("write schedule %d: %w", scheduleNumber, err)
			}
		}
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}
