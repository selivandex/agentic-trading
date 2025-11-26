package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/macro"
	"prometheus/pkg/errors"
)

// MacroRepository implements macro.Repository for ClickHouse
type MacroRepository struct {
	conn driver.Conn
}

// NewMacroRepository creates a new macro data repository
func NewMacroRepository(conn driver.Conn) *MacroRepository {
	return &MacroRepository{conn: conn}
}

// InsertEvent inserts an economic event
func (r *MacroRepository) InsertEvent(ctx context.Context, event *macro.MacroEvent) error {
	query := `
		INSERT INTO macro_events (
			id, event_type, title, country, currency, event_time,
			previous, forecast, actual, impact,
			btc_reaction_1h, btc_reaction_24h, collected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.conn.Exec(ctx, query,
		event.ID,
		string(event.EventType),
		event.Title,
		event.Country,
		event.Currency,
		event.EventTime,
		event.Previous,
		event.Forecast,
		event.Actual,
		string(event.Impact),
		event.BTCReaction1h,
		event.BTCReaction24h,
		event.CollectedAt,
	)

	if err != nil {
		return errors.Wrap(err, "insert macro event")
	}

	return nil
}

// GetUpcomingEvents retrieves upcoming economic events in a time range
func (r *MacroRepository) GetUpcomingEvents(ctx context.Context, from time.Time, to time.Time) ([]macro.MacroEvent, error) {
	query := `
		SELECT 
			id, event_type, title, country, currency, event_time,
			previous, forecast, actual, impact,
			btc_reaction_1h, btc_reaction_24h, collected_at
		FROM macro_events
		WHERE event_time >= ? AND event_time <= ?
		ORDER BY event_time ASC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "query upcoming events")
	}
	defer rows.Close()

	var events []macro.MacroEvent
	for rows.Next() {
		var e macro.MacroEvent
		var eventType, impact string

		if err := rows.Scan(
			&e.ID, &eventType, &e.Title, &e.Country, &e.Currency, &e.EventTime,
			&e.Previous, &e.Forecast, &e.Actual, &impact,
			&e.BTCReaction1h, &e.BTCReaction24h, &e.CollectedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan macro event")
		}

		e.EventType = macro.EventType(eventType)
		e.Impact = macro.ImpactLevel(impact)
		events = append(events, e)
	}

	return events, nil
}

// GetEventsByType retrieves events of a specific type
func (r *MacroRepository) GetEventsByType(ctx context.Context, eventType macro.EventType, limit int) ([]macro.MacroEvent, error) {
	query := `
		SELECT 
			id, event_type, title, country, currency, event_time,
			previous, forecast, actual, impact,
			btc_reaction_1h, btc_reaction_24h, collected_at
		FROM macro_events
		WHERE event_type = ?
		ORDER BY event_time DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, string(eventType), limit)
	if err != nil {
		return nil, errors.Wrap(err, "query events by type")
	}
	defer rows.Close()

	var events []macro.MacroEvent
	for rows.Next() {
		var e macro.MacroEvent
		var eventTypeStr, impact string

		if err := rows.Scan(
			&e.ID, &eventTypeStr, &e.Title, &e.Country, &e.Currency, &e.EventTime,
			&e.Previous, &e.Forecast, &e.Actual, &impact,
			&e.BTCReaction1h, &e.BTCReaction24h, &e.CollectedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan macro event")
		}

		e.EventType = macro.EventType(eventTypeStr)
		e.Impact = macro.ImpactLevel(impact)
		events = append(events, e)
	}

	return events, nil
}

// GetHighImpactEvents retrieves high-impact events in a time range
func (r *MacroRepository) GetHighImpactEvents(ctx context.Context, from time.Time, to time.Time) ([]macro.MacroEvent, error) {
	query := `
		SELECT 
			id, event_type, title, country, currency, event_time,
			previous, forecast, actual, impact,
			btc_reaction_1h, btc_reaction_24h, collected_at
		FROM macro_events
		WHERE event_time >= ? AND event_time <= ? AND impact = 'high'
		ORDER BY event_time ASC
		LIMIT 1000
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "query high impact events")
	}
	defer rows.Close()

	var events []macro.MacroEvent
	for rows.Next() {
		var e macro.MacroEvent
		var eventType, impact string

		if err := rows.Scan(
			&e.ID, &eventType, &e.Title, &e.Country, &e.Currency, &e.EventTime,
			&e.Previous, &e.Forecast, &e.Actual, &impact,
			&e.BTCReaction1h, &e.BTCReaction24h, &e.CollectedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan macro event")
		}

		e.EventType = macro.EventType(eventType)
		e.Impact = macro.ImpactLevel(impact)
		events = append(events, e)
	}

	return events, nil
}
