package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/jordic/goics"
	"github.com/patrickmn/go-cache"
)

const (
	siteURL        = "https://portal.ufp.pt"
	calendarURI    = "/Calendario/Eventos"
	dateTimeLayout = "2006-01-02T15:04"
	calendarName   = "UFP"
)

// Cache system
// Because the calendar change only each 3 months, we set a high cache time
// Each cached items will be stored during 6 hours
var c = cache.New(6*time.Hour, time.Hour)

// Event describes a calendar event
type Event struct {
	ID    int       `json:"id"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Title string    `json:"title"`
}

// Events describes a set of Event
type Events []*Event

// UFPCalendarResponse describes HTTP response from calendar request
type UFPCalendarResponse []struct {
	ID    int    `json:"id"`
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title"`
}

// ParseUFPCalendar request calendar from UFP portal and parse it
func ParseUFPCalendar(calendarID string) (Events, error) {
	// Search in cache
	if x, found := c.Get(calendarID); found {
		logrus.Printf("[%s] found events in cache", calendarID)
		return *x.(*Events), nil
	}

	httpClient := http.Client{
		Timeout: time.Second * 10,
	}

	// Make request
	resp, err := httpClient.Get(fmt.Sprintf("%s%s/%s.txt", siteURL, calendarURI, calendarID))
	if err != nil {
		return nil, err
	}

	// Ensure success
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	// Parse response
	var ufpCalendar UFPCalendarResponse
	if err := json.NewDecoder(resp.Body).Decode(&ufpCalendar); err != nil {
		return nil, err
	}

	// Cast calendar response into event struct
	var events = make(Events, len(ufpCalendar))
	for i, e := range ufpCalendar {
		start, err := time.Parse(dateTimeLayout, e.Start)
		if err != nil {
			return nil, err
		}

		end, err := time.Parse(dateTimeLayout, e.End)
		if err != nil {
			return nil, err
		}

		events[i] = &Event{
			ID:    e.ID,
			Start: start,
			End:   end,
			Title: e.Title,
		}
	}

	// Set events in cache
	logrus.Debugf("[%s] store events in cache", calendarID)
	c.Set(calendarID, &events, cache.DefaultExpiration)
	return events, nil
}

// EmitICal implements the interface for goics
func (e Events) EmitICal() goics.Componenter {
	c := goics.NewComponent()

	// https://tools.ietf.org/html/rfc5545
	c.SetType("VCALENDAR")
	c.AddProperty("CALSCAL", "GREGORIAN")
	c.AddProperty("PRODID", "-//UFP//NONSGML Event Calendar//PT")
	c.AddProperty("NAME", calendarName)
	c.AddProperty("X-WR-CALNAME", calendarName)
	c.AddProperty("X-WR-TIMEZONE", "UTC")
	c.AddProperty("VERSION", "2.0")

	// Required for DTSTAMP
	now := time.Now()

	for _, ev := range e {
		s := goics.NewComponent()
		s.SetType("VEVENT")

		// Dates
		k, v := goics.FormatDateTime("DTEND", ev.End)
		s.AddProperty(k, v)
		k, v = goics.FormatDateTime("DTSTART", ev.Start)
		s.AddProperty(k, v)
		k, v = goics.FormatDateTime("DTSTAMP", now)
		s.AddProperty(k, v)

		s.AddProperty("SUMMARY", ev.Title)
		s.AddProperty("UID", uuid.New().String())

		// Append to calendar
		c.AddComponent(s)
	}

	return c
}

// RenderICalHandler render calendar and response it
func RenderICalHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c, err := ParseUFPCalendar(vars["id"])
	if err != nil {
		logrus.Errorln(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Setup headers for the calendar
	w.Header().Set("Content-type", "text/calendar")
	w.Header().Set("charset", "utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("filename", "calendar.ics")
	w.WriteHeader(http.StatusOK)
	goics.NewICalEncode(w).Encode(c)
}
