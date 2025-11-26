package evaluation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// JournalCompiler compiles trading journal entries from recent trades
// Runs every hour to create journal entries for closed positions
type JournalCompiler struct {
	*workers.BaseWorker
	userRepo       user.Repository
	posRepo        position.Repository
	journalRepo    journal.Repository
	kafka          *kafka.Producer
	eventPublisher *events.WorkerPublisher
}

// NewJournalCompiler creates a new journal compiler worker
func NewJournalCompiler(
	userRepo user.Repository,
	posRepo position.Repository,
	journalRepo journal.Repository,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *JournalCompiler {
	return &JournalCompiler{
		BaseWorker:     workers.NewBaseWorker("journal_compiler", interval, enabled),
		userRepo:       userRepo,
		posRepo:        posRepo,
		journalRepo:    journalRepo,
		kafka:          kafka,
		eventPublisher: events.NewWorkerPublisher(kafka),
	}
}

// Run executes one iteration of journal compilation
func (jc *JournalCompiler) Run(ctx context.Context) error {
	jc.Log().Debug("Journal compiler: starting compilation")

	// Get all active users (using large limit)
	users, err := jc.userRepo.List(ctx, 1000, 0)
	if err != nil {
		return errors.Wrap(err, "failed to list users")
	}

	// Filter only active users
	var activeUsers []*user.User
	for _, usr := range users {
		if usr.IsActive {
			activeUsers = append(activeUsers, usr)
		}
	}
	users = activeUsers

	if len(users) == 0 {
		jc.Log().Debug("No active users for journal compilation")
		return nil
	}

	totalEntries := 0

	// Compile journals for each user
	for _, usr := range users {
		entries, err := jc.compileUserJournal(ctx, usr)
		if err != nil {
			jc.Log().Error("Failed to compile user journal",
				"user_id", usr.ID,
				"error", err,
			)
			// Continue with other users
			continue
		}

		totalEntries += entries
	}

	jc.Log().Debug("Journal compilation complete", "total_entries", totalEntries)

	return nil
}

// compileUserJournal compiles journal entries for a single user
func (jc *JournalCompiler) compileUserJournal(ctx context.Context, usr *user.User) (int, error) {
	jc.Log().Debug("Compiling user journal", "user_id", usr.ID)

	// Get closed positions from last hour that don't have journal entries yet
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	now := time.Now()

	closedPositions, err := jc.posRepo.GetClosedInRange(ctx, usr.ID, oneHourAgo, now)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get closed positions")
	}

	if len(closedPositions) == 0 {
		jc.Log().Debug("No closed positions to journal", "user_id", usr.ID)
		return 0, nil
	}

	entriesCreated := 0

	// Create journal entry for each closed position
	for _, pos := range closedPositions {
		// Check if journal entry already exists for this position
		exists, err := jc.journalRepo.ExistsForTrade(ctx, pos.ID)
		if err != nil {
			jc.Log().Error("Failed to check if journal exists",
				"position_id", pos.ID,
				"error", err,
			)
			continue
		}

		if exists {
			jc.Log().Debug("Journal entry already exists",
				"position_id", pos.ID,
			)
			continue
		}

		// Create journal entry
		if err := jc.createJournalEntry(ctx, usr.ID, pos); err != nil {
			jc.Log().Error("Failed to create journal entry",
				"position_id", pos.ID,
				"error", err,
			)
			continue
		}

		entriesCreated++
	}

	jc.Log().Info("User journal compiled",
		"user_id", usr.ID,
		"entries_created", entriesCreated,
	)

	return entriesCreated, nil
}

// createJournalEntry creates a journal entry for a closed position
func (jc *JournalCompiler) createJournalEntry(ctx context.Context, userID uuid.UUID, pos *position.Position) error {
	// Calculate hold duration in seconds
	var holdDurationSeconds int64
	if pos.ClosedAt != nil {
		holdDurationSeconds = int64(pos.ClosedAt.Sub(pos.OpenedAt).Seconds())
	}

	// Determine if entry was correct (simplified - in production use more sophisticated logic)
	wasCorrectEntry := pos.RealizedPnL.IsPositive()
	wasCorrectExit := pos.RealizedPnL.IsPositive()

	// Create journal entry with correct fields
	entry := &journal.JournalEntry{
		UserID:  userID,
		TradeID: pos.ID,
		Symbol:  pos.Symbol,
		Side:    pos.Side.String(),

		EntryPrice: pos.EntryPrice,
		ExitPrice:  pos.CurrentPrice, // Use current price as exit price
		Size:       pos.Size,
		PnL:        pos.RealizedPnL,
		PnLPercent: pos.UnrealizedPnLPct, // Best approximation we have

		// Strategy info (from trading pair, not directly on position)
		StrategyUsed: "unknown", // TODO: Get from trading pair
		Timeframe:    "unknown",
		SetupType:    "unknown",

		// Decision context
		MarketRegime:    "unknown", // TODO: Get current regime
		EntryReasoning:  pos.OpenReasoning,
		ExitReasoning:   "position_closed",
		ConfidenceScore: 0.0, // TODO: Get from agent decision

		// Indicators at entry (zeros for now, TODO: capture at entry time)
		RSIAtEntry:    0,
		ATRAtEntry:    0,
		VolumeAtEntry: 0,

		// Outcome analysis
		WasCorrectEntry: wasCorrectEntry,
		WasCorrectExit:  wasCorrectExit,
		MaxDrawdown:     pos.UnrealizedPnL.Neg(), // Approximation
		MaxProfit:       pos.RealizedPnL,
		HoldDuration:    holdDurationSeconds,

		// Lessons learned (AI generated - TODO)
		LessonsLearned:  jc.generateTradeNotes(pos),
		ImprovementTips: "",

		CreatedAt: time.Now(),
	}

	// Save journal entry
	if err := jc.journalRepo.Create(ctx, entry); err != nil {
		return errors.Wrap(err, "failed to save journal entry")
	}

	jc.Log().Debug("Journal entry created",
		"position_id", pos.ID,
		"symbol", pos.Symbol,
		"pnl", pos.RealizedPnL,
	)

	// Publish journal entry event
	jc.publishJournalEvent(ctx, entry)

	return nil
}

// generateTradeNotes generates automatic notes for a trade
func (jc *JournalCompiler) generateTradeNotes(pos *position.Position) string {
	notes := ""

	// Add outcome note
	if pos.RealizedPnL.IsPositive() {
		notes += "✅ Winning trade. "
	} else if pos.RealizedPnL.IsNegative() {
		notes += "❌ Losing trade. "
	} else {
		notes += "➖ Break-even trade. "
	}

	// Add SL/TP notes
	if !pos.StopLossPrice.IsZero() {
		notes += "SL was set. "
	}
	if !pos.TakeProfitPrice.IsZero() {
		notes += "TP was set. "
	}

	// TODO: Add more contextual notes:
	// - Market regime at entry
	// - Agent confidence scores
	// - Indicators that triggered entry
	// - Exit reason (SL hit, TP hit, manual, etc)

	return notes
}

// Event structure

type JournalEntryEvent struct {
	UserID         string    `json:"user_id"`
	TradeID        string    `json:"trade_id"`
	Symbol         string    `json:"symbol"`
	Strategy       string    `json:"strategy"`
	Outcome        string    `json:"outcome"`
	RealizedPnL    string    `json:"realized_pnl"`
	RealizedPnLPct string    `json:"realized_pnl_pct"`
	HoldDuration   string    `json:"hold_duration"`
	Notes          string    `json:"notes"`
	Timestamp      time.Time `json:"timestamp"`
}

func (jc *JournalCompiler) publishJournalEvent(ctx context.Context, entry *journal.JournalEntry) {
	pnl, _ := entry.PnL.Float64()
	pnlPercent, _ := entry.PnLPercent.Float64()

	sideStr := string(entry.Side)

	if err := jc.eventPublisher.PublishJournalEntryCreated(
		ctx,
		entry.UserID.String(),
		entry.ID.String(),
		entry.Symbol,
		sideStr,
		entry.LessonsLearned,
		pnl,
		pnlPercent,
	); err != nil {
		jc.Log().Error("Failed to publish journal entry event", "error", err)
	}
}
