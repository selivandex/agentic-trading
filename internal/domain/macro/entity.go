package macro

import "time"

// MacroEvent represents an economic calendar event
type MacroEvent struct {
	ID        string      `db:"id"`
	EventType EventType   `db:"event_type"`
	Title     string      `db:"title"`
	Country   string      `db:"country"`
	Currency  string      `db:"currency"`

	// Timing
	EventTime time.Time `db:"event_time"`

	// Values
	Previous string `db:"previous"`
	Forecast string `db:"forecast"`
	Actual   string `db:"actual"`

	// Impact
	Impact ImpactLevel `db:"impact"`

	// Historical price reaction
	BTCReaction1h  float64 `db:"btc_reaction_1h"`  // % change 1h after
	BTCReaction24h float64 `db:"btc_reaction_24h"` // % change 24h after

	CollectedAt time.Time `db:"collected_at"`
}

// EventType defines economic event types
type EventType string

const (
	EventCPI          EventType = "cpi"
	EventPPI          EventType = "ppi"
	EventFOMC         EventType = "fomc"
	EventNFP          EventType = "nfp"
	EventGDP          EventType = "gdp"
	EventPMI          EventType = "pmi"
	EventUnemployment EventType = "unemployment"
	EventRetailSales  EventType = "retail_sales"
	EventFedSpeech    EventType = "fed_speech"
)

// Valid checks if event type is valid
func (e EventType) Valid() bool {
	switch e {
	case EventCPI, EventPPI, EventFOMC, EventNFP, EventGDP,
		EventPMI, EventUnemployment, EventRetailSales, EventFedSpeech:
		return true
	}
	return false
}

// String returns string representation
func (e EventType) String() string {
	return string(e)
}

// ImpactLevel defines event impact level
type ImpactLevel string

const (
	ImpactLow    ImpactLevel = "low"
	ImpactMedium ImpactLevel = "medium"
	ImpactHigh   ImpactLevel = "high"
)

// Valid checks if impact level is valid
func (i ImpactLevel) Valid() bool {
	switch i {
	case ImpactLow, ImpactMedium, ImpactHigh:
		return true
	}
	return false
}

// String returns string representation
func (i ImpactLevel) String() string {
	return string(i)
}


import "time"

// MacroEvent represents an economic calendar event
type MacroEvent struct {
	ID        string      `db:"id"`
	EventType EventType   `db:"event_type"`
	Title     string      `db:"title"`
	Country   string      `db:"country"`
	Currency  string      `db:"currency"`

	// Timing
	EventTime time.Time `db:"event_time"`

	// Values
	Previous string `db:"previous"`
	Forecast string `db:"forecast"`
	Actual   string `db:"actual"`

	// Impact
	Impact ImpactLevel `db:"impact"`

	// Historical price reaction
	BTCReaction1h  float64 `db:"btc_reaction_1h"`  // % change 1h after
	BTCReaction24h float64 `db:"btc_reaction_24h"` // % change 24h after

	CollectedAt time.Time `db:"collected_at"`
}

// EventType defines economic event types
type EventType string

const (
	EventCPI          EventType = "cpi"
	EventPPI          EventType = "ppi"
	EventFOMC         EventType = "fomc"
	EventNFP          EventType = "nfp"
	EventGDP          EventType = "gdp"
	EventPMI          EventType = "pmi"
	EventUnemployment EventType = "unemployment"
	EventRetailSales  EventType = "retail_sales"
	EventFedSpeech    EventType = "fed_speech"
)

// Valid checks if event type is valid
func (e EventType) Valid() bool {
	switch e {
	case EventCPI, EventPPI, EventFOMC, EventNFP, EventGDP,
		EventPMI, EventUnemployment, EventRetailSales, EventFedSpeech:
		return true
	}
	return false
}

// String returns string representation
func (e EventType) String() string {
	return string(e)
}

// ImpactLevel defines event impact level
type ImpactLevel string

const (
	ImpactLow    ImpactLevel = "low"
	ImpactMedium ImpactLevel = "medium"
	ImpactHigh   ImpactLevel = "high"
)

// Valid checks if impact level is valid
func (i ImpactLevel) Valid() bool {
	switch i {
	case ImpactLow, ImpactMedium, ImpactHigh:
		return true
	}
	return false
}

// String returns string representation
func (i ImpactLevel) String() string {
	return string(i)
}

