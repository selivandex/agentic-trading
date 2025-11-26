package macro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"prometheus/internal/domain/macro"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// EconomicCalendarCollector collects economic events from various sources
// Tracks CPI, NFP, FOMC meetings, GDP releases, etc.
type EconomicCalendarCollector struct {
	*workers.BaseWorker
	macroRepo  macro.Repository
	httpClient *http.Client
	apiKey     string
	countries  []string
	eventTypes []macro.EventType
}

// NewEconomicCalendarCollector creates a new economic calendar collector
func NewEconomicCalendarCollector(
	macroRepo macro.Repository,
	apiKey string,
	countries []string,
	eventTypes []macro.EventType,
	interval time.Duration,
	enabled bool,
) *EconomicCalendarCollector {
	return &EconomicCalendarCollector{
		BaseWorker: workers.NewBaseWorker("economic_calendar_collector", interval, enabled),
		macroRepo:  macroRepo,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		countries:  countries,
		eventTypes: eventTypes,
	}
}

// Run executes one iteration of economic calendar collection
func (ec *EconomicCalendarCollector) Run(ctx context.Context) error {
	ec.Log().Debug("Economic calendar collector: starting iteration")

	if ec.apiKey == "" {
		ec.Log().Debug("Trading Economics API key not configured, skipping iteration")
		return nil
	}

	// Fetch upcoming economic events
	events, err := ec.fetchEconomicEvents(ctx)
	if err != nil {
		ec.Log().Error("Failed to fetch economic events", "error", err)
		return errors.Wrap(err, "fetch economic events")
	}

	if len(events) == 0 {
		ec.Log().Debug("No economic events found")
		return nil
	}

	// Store events in database
	savedCount := 0
	for _, event := range events {
		if err := ec.macroRepo.InsertEvent(ctx, &event); err != nil {
			ec.Log().Error("Failed to save economic event",
				"event", event.Title,
				"error", err,
			)
			continue
		}
		savedCount++
	}

	ec.Log().Info("Economic events collected",
		"total", len(events),
		"saved", savedCount,
	)

	return nil
}

// Trading Economics API response structures
// Note: Actual API structure may differ - this is a placeholder
type tradingEconomicsResponse struct {
	Events []tradingEconomicsEvent `json:"events"`
}

type tradingEconomicsEvent struct {
	CalendarID string `json:"CalendarId"`
	Date       string `json:"Date"`
	Country    string `json:"Country"`
	Category   string `json:"Category"`
	Event      string `json:"Event"`
	Reference  string `json:"Reference"`
	Source     string `json:"Source"`
	Actual     string `json:"Actual"`
	Previous   string `json:"Previous"`
	Forecast   string `json:"Forecast"`
	TEForecast string `json:"TEForecast"`
	Importance int    `json:"Importance"` // 1=low, 2=medium, 3=high
	LastUpdate string `json:"LastUpdate"`
	Revised    string `json:"Revised"`
}

// fetchEconomicEvents fetches upcoming events from Trading Economics API
func (ec *EconomicCalendarCollector) fetchEconomicEvents(ctx context.Context) ([]macro.MacroEvent, error) {
	// Trading Economics API endpoint
	// Docs: https://docs.tradingeconomics.com/

	// Get events for next 7 days
	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	url := fmt.Sprintf("https://api.tradingeconomics.com/calendar?c=%s&d1=%s&d2=%s&key=%s",
		strings.Join(ec.countries, ","),
		startDate,
		endDate,
		ec.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Trading Economics API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Trading Economics API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		ec.Log().Warn("Trading Economics API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Trading Economics API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiEvents []tradingEconomicsEvent
	if err := json.NewDecoder(resp.Body).Decode(&apiEvents); err != nil {
		return nil, errors.Wrap(err, "decode Trading Economics API response")
	}

	ec.Log().Debug("Fetched economic events from API", "count", len(apiEvents))

	// Convert to internal format
	var events []macro.MacroEvent
	for _, apiEvent := range apiEvents {
		event := ec.convertEvent(apiEvent)
		if event != nil {
			// Filter by event type if configured
			if len(ec.eventTypes) > 0 && !ec.containsEventType(event.EventType) {
				continue
			}
			events = append(events, *event)
		}
	}

	return events, nil
}

// convertEvent converts Trading Economics event to internal format
func (ec *EconomicCalendarCollector) convertEvent(apiEvent tradingEconomicsEvent) *macro.MacroEvent {
	// Parse event time
	eventTime, err := time.Parse("2006-01-02T15:04:05", apiEvent.Date)
	if err != nil {
		// Try alternative format
		eventTime, err = time.Parse("2006-01-02", apiEvent.Date)
		if err != nil {
			ec.Log().Warn("Failed to parse event time", "date", apiEvent.Date)
			eventTime = time.Now()
		}
	}

	// Determine event type from category/event name
	eventType := ec.determineEventType(apiEvent.Category, apiEvent.Event)
	if !eventType.Valid() {
		// Skip unknown event types
		return nil
	}

	// Map importance to impact level
	impact := macro.ImpactLow
	switch apiEvent.Importance {
	case 3:
		impact = macro.ImpactHigh
	case 2:
		impact = macro.ImpactMedium
	default:
		impact = macro.ImpactLow
	}

	// Determine currency from country
	currency := ec.countryCurrency(apiEvent.Country)

	return &macro.MacroEvent{
		ID:          apiEvent.CalendarID,
		EventType:   eventType,
		Title:       apiEvent.Event,
		Country:     apiEvent.Country,
		Currency:    currency,
		EventTime:   eventTime,
		Previous:    apiEvent.Previous,
		Forecast:    apiEvent.Forecast,
		Actual:      apiEvent.Actual,
		Impact:      impact,
		CollectedAt: time.Now(),
	}
}

// determineEventType maps event category/name to EventType
func (ec *EconomicCalendarCollector) determineEventType(category, event string) macro.EventType {
	eventLower := strings.ToLower(event + " " + category)

	if strings.Contains(eventLower, "cpi") || strings.Contains(eventLower, "inflation") {
		return macro.EventCPI
	}
	if strings.Contains(eventLower, "ppi") || strings.Contains(eventLower, "producer price") {
		return macro.EventPPI
	}
	if strings.Contains(eventLower, "fomc") || strings.Contains(eventLower, "federal reserve") {
		return macro.EventFOMC
	}
	if strings.Contains(eventLower, "nfp") || strings.Contains(eventLower, "nonfarm payroll") {
		return macro.EventNFP
	}
	if strings.Contains(eventLower, "gdp") || strings.Contains(eventLower, "gross domestic") {
		return macro.EventGDP
	}
	if strings.Contains(eventLower, "pmi") || strings.Contains(eventLower, "purchasing managers") {
		return macro.EventPMI
	}
	if strings.Contains(eventLower, "unemployment") || strings.Contains(eventLower, "jobless") {
		return macro.EventUnemployment
	}
	if strings.Contains(eventLower, "retail sales") {
		return macro.EventRetailSales
	}
	if strings.Contains(eventLower, "fed") && (strings.Contains(eventLower, "speech") || strings.Contains(eventLower, "testimony")) {
		return macro.EventFedSpeech
	}

	// Unknown event type
	return ""
}

// countryCurrency maps country to currency
func (ec *EconomicCalendarCollector) countryCurrency(country string) string {
	currencyMap := map[string]string{
		"United States":  "USD",
		"Euro Area":      "EUR",
		"United Kingdom": "GBP",
		"Japan":          "JPY",
		"China":          "CNY",
		"Switzerland":    "CHF",
		"Canada":         "CAD",
		"Australia":      "AUD",
	}

	if currency, ok := currencyMap[country]; ok {
		return currency
	}

	return "USD" // Default
}

// containsEventType checks if an event type is in the configured list
func (ec *EconomicCalendarCollector) containsEventType(eventType macro.EventType) bool {
	for _, et := range ec.eventTypes {
		if et == eventType {
			return true
		}
	}
	return false
}

// CalculateEventSurprise calculates the surprise index for an event
// Surprise = (Actual - Forecast) / |Forecast|
func CalculateEventSurprise(event *macro.MacroEvent) float64 {
	if event.Actual == "" || event.Forecast == "" {
		return 0
	}

	// Parse values (handle different formats: "3.2%", "199K", "2.4")
	actual := parseEconomicValue(event.Actual)
	forecast := parseEconomicValue(event.Forecast)

	if forecast == 0 {
		return 0
	}

	surprise := (actual - forecast) / forecast
	return surprise
}

// parseEconomicValue parses economic indicator values
func parseEconomicValue(value string) float64 {
	// Remove % sign
	value = strings.TrimSuffix(value, "%")

	// Remove K/M/B suffixes and convert
	multiplier := 1.0
	if strings.HasSuffix(value, "K") {
		multiplier = 1000
		value = strings.TrimSuffix(value, "K")
	} else if strings.HasSuffix(value, "M") {
		multiplier = 1_000_000
		value = strings.TrimSuffix(value, "M")
	} else if strings.HasSuffix(value, "B") {
		multiplier = 1_000_000_000
		value = strings.TrimSuffix(value, "B")
	}

	// Parse float
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}

	return parsed * multiplier
}
