package telegram

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"prometheus/pkg/logger"
)

// AsyncMessageQueue manages async message sending with rate limiting
type AsyncMessageQueue struct {
	bot            Bot
	queue          chan *queuedMessage
	workers        int
	rateLimitDelay time.Duration
	log            *logger.Logger
	wg             sync.WaitGroup
	stopCh         chan struct{}
	mu             sync.Mutex
	running        bool
}

// queuedMessage represents a message in the queue
type queuedMessage struct {
	chatID   int64
	text     string
	opts     MessageOptions
	callback func(messageID int, err error)
}

// NewAsyncMessageQueue creates a new async message queue
func NewAsyncMessageQueue(bot Bot, workers int, rateLimitDelay time.Duration, log *logger.Logger) *AsyncMessageQueue {
	if workers <= 0 {
		workers = 5 // Default 5 workers
	}
	if rateLimitDelay == 0 {
		rateLimitDelay = 50 * time.Millisecond // Default 50ms between messages
	}

	return &AsyncMessageQueue{
		bot:            bot,
		queue:          make(chan *queuedMessage, 1000), // Buffer 1000 messages
		workers:        workers,
		rateLimitDelay: rateLimitDelay,
		log:            log.With("component", "async_message_queue"),
		stopCh:         make(chan struct{}),
	}
}

// Start starts queue workers
func (amq *AsyncMessageQueue) Start() {
	amq.mu.Lock()
	defer amq.mu.Unlock()

	if amq.running {
		amq.log.Warnw("Async message queue already running")
		return
	}

	amq.running = true
	amq.log.Infow("Starting async message queue", "workers", amq.workers)

	for i := 0; i < amq.workers; i++ {
		amq.wg.Add(1)
		go amq.worker(i)
	}
}

// Stop stops queue workers gracefully
func (amq *AsyncMessageQueue) Stop() {
	amq.mu.Lock()
	defer amq.mu.Unlock()

	if !amq.running {
		return
	}

	amq.log.Infow("Stopping async message queue")
	close(amq.stopCh)
	amq.wg.Wait()
	amq.running = false
	amq.log.Infow("Async message queue stopped")
}

// Enqueue adds message to queue for async sending
func (amq *AsyncMessageQueue) Enqueue(chatID int64, text string, opts MessageOptions, callback func(messageID int, err error)) error {
	select {
	case amq.queue <- &queuedMessage{
		chatID:   chatID,
		text:     text,
		opts:     opts,
		callback: callback,
	}:
		return nil
	case <-amq.stopCh:
		return ErrQueueStopped
	default:
		amq.log.Warnw("Message queue full, message dropped",
			"chat_id", chatID,
			"queue_size", len(amq.queue),
		)
		return ErrQueueFull
	}
}

// worker processes messages from queue
func (amq *AsyncMessageQueue) worker(id int) {
	defer amq.wg.Done()

	amq.log.Debugw("Worker started", "worker_id", id)

	for {
		select {
		case msg := <-amq.queue:
			amq.processMessage(msg, id)
			// Rate limiting delay
			time.Sleep(amq.rateLimitDelay)

		case <-amq.stopCh:
			amq.log.Debugw("Worker stopping", "worker_id", id)
			return
		}
	}
}

// processMessage sends a single message
func (amq *AsyncMessageQueue) processMessage(msg *queuedMessage, workerID int) {
	amq.log.Debugw("Processing queued message",
		"worker_id", workerID,
		"chat_id", msg.chatID,
	)

	// Send message
	messageID, err := amq.bot.SendMessageWithOptions(msg.chatID, msg.text, msg.opts)

	if err != nil {
		amq.log.Errorw("Failed to send queued message",
			"worker_id", workerID,
			"chat_id", msg.chatID,
			"error", err,
		)
	}

	// Call callback if provided
	if msg.callback != nil {
		msg.callback(messageID, err)
	}

	// Handle self-destruct
	if err == nil && msg.opts.SelfDestruct > 0 {
		amq.scheduleSelfDestruct(msg.chatID, messageID, msg.opts.SelfDestruct)
	}
}

// scheduleSelfDestruct schedules message deletion
func (amq *AsyncMessageQueue) scheduleSelfDestruct(chatID int64, messageID int, duration time.Duration) {
	go func() {
		time.Sleep(duration)
		amq.log.Debugw("Self-destructing message",
			"chat_id", chatID,
			"message_id", messageID,
			"duration", duration,
		)
		if err := amq.bot.DeleteMessage(chatID, messageID); err != nil {
			amq.log.Errorw("Failed to self-destruct message",
				"chat_id", chatID,
				"message_id", messageID,
				"error", err,
			)
		}
	}()
}

// GetQueueSize returns current queue size
func (amq *AsyncMessageQueue) GetQueueSize() int {
	return len(amq.queue)
}

// IsRunning checks if queue is running
func (amq *AsyncMessageQueue) IsRunning() bool {
	amq.mu.Lock()
	defer amq.mu.Unlock()
	return amq.running
}

// SelfDestructingMessage sends a message that auto-deletes after duration
type SelfDestructingMessage struct {
	bot      *tgbotapi.BotAPI
	chatID   int64
	text     string
	duration time.Duration
	log      *logger.Logger
}

// Send sends a self-destructing message
func (sdm *SelfDestructingMessage) Send() error {
	msg := tgbotapi.NewMessage(sdm.chatID, sdm.text)
	sentMsg, err := sdm.bot.Send(msg)
	if err != nil {
		return err
	}

	// Schedule deletion
	go func() {
		time.Sleep(sdm.duration)
		deleteMsg := tgbotapi.NewDeleteMessage(sdm.chatID, sentMsg.MessageID)
		if _, err := sdm.bot.Request(deleteMsg); err != nil {
			sdm.log.Errorw("Failed to delete self-destructing message",
				"chat_id", sdm.chatID,
				"message_id", sentMsg.MessageID,
				"error", err,
			)
		} else {
			sdm.log.Debugw("Self-destructing message deleted",
				"chat_id", sdm.chatID,
				"message_id", sentMsg.MessageID,
			)
		}
	}()

	return nil
}

// Common errors
var (
	ErrQueueFull    = &BotError{Code: "QUEUE_FULL", Message: "Message queue is full"}
	ErrQueueStopped = &BotError{Code: "QUEUE_STOPPED", Message: "Message queue is stopped"}
)

// BotError represents a telegram bot error
type BotError struct {
	Code    string
	Message string
}

// Error implements error interface
func (e *BotError) Error() string {
	return e.Message
}
