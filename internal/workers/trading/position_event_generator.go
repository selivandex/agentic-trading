package trading

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

// PositionEventGenerator checks position triggers and generates events
type PositionEventGenerator struct {
	publisher *events.Publisher
	log       *logger.Logger

	// Configuration thresholds
	stopApproachingThreshold   float64 // 2% by default
	targetApproachingThreshold float64 // 5% by default
	profitMilestones           []int32 // [5, 10, 20, 50, 100]
	maxPositionDurationHours   int64   // 72 hours by default (3 days)
}

// NewPositionEventGenerator creates a new position event generator
func NewPositionEventGenerator(publisher *events.Publisher, log *logger.Logger) *PositionEventGenerator {
	return &PositionEventGenerator{
		publisher:                  publisher,
		log:                        log,
		stopApproachingThreshold:   2.0, // 2%
		targetApproachingThreshold: 5.0, // 5%
		profitMilestones:           []int32{5, 10, 20, 50, 100},
		maxPositionDurationHours:   72, // 3 days
	}
}

// CheckPositionTriggers checks all triggers for a position and publishes events
func (g *PositionEventGenerator) CheckPositionTriggers(
	ctx context.Context,
	pos *position.Position,
	currentPrice decimal.Decimal,
	exchangeName string,
) error {
	// Check stop loss proximity
	if !pos.StopLossPrice.IsZero() {
		if err := g.checkStopProximity(ctx, pos, currentPrice, exchangeName); err != nil {
			g.log.Error("Failed to check stop proximity", "error", err)
		}
	}

	// Check take profit proximity
	if !pos.TakeProfitPrice.IsZero() {
		if err := g.checkTargetProximity(ctx, pos, currentPrice, exchangeName); err != nil {
			g.log.Error("Failed to check target proximity", "error", err)
		}
	}

	// Check profit milestones
	if err := g.checkProfitMilestones(ctx, pos, currentPrice, exchangeName); err != nil {
		g.log.Error("Failed to check profit milestones", "error", err)
	}

	// Check time decay
	if err := g.checkTimeDecay(ctx, pos, exchangeName); err != nil {
		g.log.Error("Failed to check time decay", "error", err)
	}

	return nil
}

// checkStopProximity checks if price is approaching stop loss
func (g *PositionEventGenerator) checkStopProximity(
	ctx context.Context,
	pos *position.Position,
	currentPrice decimal.Decimal,
	exchangeName string,
) error {
	var distancePercent decimal.Decimal
	var approaching bool

	if pos.Side == position.PositionLong {
		// Long: stop below current price
		if currentPrice.GreaterThan(pos.StopLossPrice) {
			distance := currentPrice.Sub(pos.StopLossPrice)
			distancePercent = distance.Div(currentPrice).Mul(decimal.NewFromInt(100))
			approaching = distancePercent.LessThanOrEqual(decimal.NewFromFloat(g.stopApproachingThreshold))
		}
	} else {
		// Short: stop above current price
		if currentPrice.LessThan(pos.StopLossPrice) {
			distance := pos.StopLossPrice.Sub(currentPrice)
			distancePercent = distance.Div(currentPrice).Mul(decimal.NewFromInt(100))
			approaching = distancePercent.LessThanOrEqual(decimal.NewFromFloat(g.stopApproachingThreshold))
		}
	}

	if approaching {
		distPct, _ := distancePercent.Float64()
		currentPriceF, _ := currentPrice.Float64()
		stopPriceF, _ := pos.StopLossPrice.Float64()
		entryPriceF, _ := pos.EntryPrice.Float64()

		event := &eventspb.StopApproachingEvent{
			Base:            events.NewBaseEvent("stop_approaching", "position_monitor", pos.UserID.String()),
			PositionId:      pos.ID.String(),
			Symbol:          pos.Symbol,
			Exchange:        exchangeName,
			CurrentPrice:    currentPriceF,
			StopLossPrice:   stopPriceF,
			DistancePercent: distPct,
			Urgency:         eventspb.EventUrgency_HIGH,
			EntryPrice:      entryPriceF,
			Side:            pos.Side.String(),
		}

		if err := g.publisher.PublishStopApproaching(ctx, event); err != nil {
			return err
		}

		g.log.Info("Stop approaching event published",
			"position_id", pos.ID,
			"symbol", pos.Symbol,
			"distance_pct", distPct,
		)
	}

	return nil
}

// checkTargetProximity checks if price is approaching take profit
func (g *PositionEventGenerator) checkTargetProximity(
	ctx context.Context,
	pos *position.Position,
	currentPrice decimal.Decimal,
	exchangeName string,
) error {
	var distancePercent decimal.Decimal
	var approaching bool

	if pos.Side == position.PositionLong {
		// Long: target above current price
		if currentPrice.LessThan(pos.TakeProfitPrice) {
			distance := pos.TakeProfitPrice.Sub(currentPrice)
			distancePercent = distance.Div(currentPrice).Mul(decimal.NewFromInt(100))
			approaching = distancePercent.LessThanOrEqual(decimal.NewFromFloat(g.targetApproachingThreshold))
		}
	} else {
		// Short: target below current price
		if currentPrice.GreaterThan(pos.TakeProfitPrice) {
			distance := currentPrice.Sub(pos.TakeProfitPrice)
			distancePercent = distance.Div(currentPrice).Mul(decimal.NewFromInt(100))
			approaching = distancePercent.LessThanOrEqual(decimal.NewFromFloat(g.targetApproachingThreshold))
		}
	}

	if approaching {
		distPct, _ := distancePercent.Float64()
		currentPriceF, _ := currentPrice.Float64()
		targetPriceF, _ := pos.TakeProfitPrice.Float64()

		// Calculate unrealized PnL percent
		var pnlPct decimal.Decimal
		if pos.Side == position.PositionLong {
			pnlPct = currentPrice.Sub(pos.EntryPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
		} else {
			pnlPct = pos.EntryPrice.Sub(currentPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
		}
		pnlPctF, _ := pnlPct.Float64()

		event := &eventspb.TargetApproachingEvent{
			Base:                 events.NewBaseEvent("target_approaching", "position_monitor", pos.UserID.String()),
			PositionId:           pos.ID.String(),
			Symbol:               pos.Symbol,
			Exchange:             exchangeName,
			CurrentPrice:         currentPriceF,
			TakeProfitPrice:      targetPriceF,
			DistancePercent:      distPct,
			Urgency:              eventspb.EventUrgency_MEDIUM,
			UnrealizedPnlPercent: pnlPctF,
			Side:                 pos.Side.String(),
		}

		if err := g.publisher.PublishTargetApproaching(ctx, event); err != nil {
			return err
		}

		g.log.Info("Target approaching event published",
			"position_id", pos.ID,
			"symbol", pos.Symbol,
			"distance_pct", distPct,
			"pnl_pct", pnlPctF,
		)
	}

	return nil
}

// checkProfitMilestones checks if position reached profit milestones
func (g *PositionEventGenerator) checkProfitMilestones(
	ctx context.Context,
	pos *position.Position,
	currentPrice decimal.Decimal,
	exchangeName string,
) error {
	// Calculate unrealized PnL percent
	var pnlPct decimal.Decimal
	if pos.Side == position.PositionLong {
		pnlPct = currentPrice.Sub(pos.EntryPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
	} else {
		pnlPct = pos.EntryPrice.Sub(currentPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
	}

	// Check if any milestone reached
	// Note: In production, you'd want to track which milestones have already been triggered
	// to avoid duplicate events. This could be done via position metadata or separate tracking table.
	for _, milestone := range g.profitMilestones {
		milestoneDecimal := decimal.NewFromInt32(milestone)

		// Check if we've just crossed this milestone (within 0.5% tolerance)
		if pnlPct.GreaterThanOrEqual(milestoneDecimal) &&
			pnlPct.LessThan(milestoneDecimal.Add(decimal.NewFromFloat(0.5))) {

			pnlPctF, _ := pnlPct.Float64()
			currentPriceF, _ := currentPrice.Float64()
			entryPriceF, _ := pos.EntryPrice.Float64()

			// Determine urgency based on milestone
			var urgency eventspb.EventUrgency
			switch milestone {
			case 5:
				urgency = eventspb.EventUrgency_MEDIUM
			case 10, 20:
				urgency = eventspb.EventUrgency_HIGH
			default:
				urgency = eventspb.EventUrgency_MEDIUM
			}

			event := &eventspb.ProfitMilestoneEvent{
				Base:                 events.NewBaseEvent("profit_milestone", "position_monitor", pos.UserID.String()),
				PositionId:           pos.ID.String(),
				Symbol:               pos.Symbol,
				Exchange:             exchangeName,
				UnrealizedPnlPercent: pnlPctF,
				Milestone:            milestone,
				CurrentPrice:         currentPriceF,
				EntryPrice:           entryPriceF,
				Urgency:              urgency,
				StopAtBreakeven:      milestone >= 5, // Recommend trailing at +5%
			}

			if err := g.publisher.PublishProfitMilestone(ctx, event); err != nil {
				return err
			}

			g.log.Info("Profit milestone event published",
				"position_id", pos.ID,
				"symbol", pos.Symbol,
				"milestone", milestone,
				"pnl_pct", pnlPctF,
			)
		}
	}

	return nil
}

// checkTimeDecay checks if position is held longer than expected
func (g *PositionEventGenerator) checkTimeDecay(
	ctx context.Context,
	pos *position.Position,
	exchangeName string,
) error {
	now := time.Now()
	duration := now.Sub(pos.OpenedAt)
	durationHours := int64(duration.Hours())

	// Only trigger if position is held significantly longer than expected
	if durationHours > g.maxPositionDurationHours {
		// Calculate current PnL percent
		var pnlPct float64
		if !pos.UnrealizedPnLPct.IsZero() {
			pnlPct, _ = pos.UnrealizedPnLPct.Float64()
		}

		// Determine urgency: if losing and held too long = HIGH, if winning = MEDIUM
		var urgency eventspb.EventUrgency
		if pnlPct < 0 {
			urgency = eventspb.EventUrgency_HIGH
		} else {
			urgency = eventspb.EventUrgency_MEDIUM
		}

		event := &eventspb.TimeDecayEvent{
			Base:                  events.NewBaseEvent("time_decay", "position_monitor", pos.UserID.String()),
			PositionId:            pos.ID.String(),
			Symbol:                pos.Symbol,
			Exchange:              exchangeName,
			DurationHours:         durationHours,
			ExpectedDurationHours: g.maxPositionDurationHours,
			CurrentPnlPercent:     pnlPct,
			Urgency:               urgency,
			Timeframe:             "unknown", // Could be stored in position metadata
		}

		if err := g.publisher.PublishTimeDecay(ctx, event); err != nil {
			return err
		}

		g.log.Info("Time decay event published",
			"position_id", pos.ID,
			"symbol", pos.Symbol,
			"duration_hours", durationHours,
			"pnl_pct", pnlPct,
		)
	}

	return nil
}
