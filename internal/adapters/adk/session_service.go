package adk

import (
	"context"
	"encoding/json"
	"iter"
	"time"

	"google.golang.org/adk/session"
	"google.golang.org/genai"

	domainsession "prometheus/internal/domain/session"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// SessionService adapts our domain session service to ADK's session.Service interface
type SessionService struct {
	domainService *domainsession.Service
	log           *logger.Logger
}

// NewSessionService creates a new ADK session service adapter
func NewSessionService(domainService *domainsession.Service) session.Service {
	return &SessionService{
		domainService: domainService,
		log:           logger.Get().With("component", "adk_session_adapter"),
	}
}

// Create creates a new session
func (s *SessionService) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	if req == nil || req.AppName == "" || req.UserID == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name and user_id are required")
	}

	// Create session using domain service
	domainSess, err := s.domainService.CreateSession(
		ctx,
		req.AppName,
		req.UserID,
		req.SessionID,
		req.State,
	)
	if err != nil {
		return nil, err
	}

	// Convert to ADK session
	adkSession := s.domainToADKSession(domainSess)

	return &session.CreateResponse{
		Session: adkSession,
	}, nil
}

// Get retrieves a session
func (s *SessionService) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	if req == nil || req.AppName == "" || req.UserID == "" || req.SessionID == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name, user_id, and session_id are required")
	}

	opts := &domainsession.GetOptions{
		NumRecentEvents: req.NumRecentEvents,
		After:           req.After,
	}

	domainSess, err := s.domainService.GetSession(ctx, req.AppName, req.UserID, req.SessionID, opts)
	if err != nil {
		return nil, err
	}

	// Convert to ADK session
	adkSession := s.domainToADKSession(domainSess)

	return &session.GetResponse{
		Session: adkSession,
	}, nil
}

// List lists sessions
func (s *SessionService) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	if req == nil || req.AppName == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name is required")
	}

	domainSessions, err := s.domainService.ListSessions(ctx, req.AppName, req.UserID)
	if err != nil {
		return nil, err
	}

	// Convert to ADK sessions
	adkSessions := make([]session.Session, len(domainSessions))
	for i, domainSess := range domainSessions {
		adkSessions[i] = s.domainToADKSession(domainSess)
	}

	return &session.ListResponse{
		Sessions: adkSessions,
	}, nil
}

// Delete deletes a session
func (s *SessionService) Delete(ctx context.Context, req *session.DeleteRequest) error {
	if req == nil || req.AppName == "" || req.UserID == "" || req.SessionID == "" {
		return errors.Wrap(errors.ErrInvalidInput, "app_name, user_id, and session_id are required")
	}

	return s.domainService.DeleteSession(ctx, req.AppName, req.UserID, req.SessionID)
}

// AppendEvent appends an event to a session
func (s *SessionService) AppendEvent(ctx context.Context, sess session.Session, event *session.Event) error {
	if sess == nil || event == nil {
		return errors.Wrap(errors.ErrInvalidInput, "session and event are required")
	}

	// Convert ADK session to domain session
	domainSess, err := s.adkToDomainSession(sess)
	if err != nil {
		return errors.Wrap(err, "failed to convert session")
	}

	// Convert ADK event to domain event
	domainEvent, err := s.adkToDomainEvent(event)
	if err != nil {
		return errors.Wrap(err, "failed to convert event")
	}

	// Append event using domain service
	return s.domainService.AppendEvent(ctx, domainSess, domainEvent)
}

// domainToADKSession converts our domain session to ADK session
func (s *SessionService) domainToADKSession(domainSess *domainsession.Session) session.Session {
	return &adkSession{
		appName:      domainSess.AppName,
		userID:       domainSess.UserID,
		sessionID:    domainSess.SessionID,
		state:        domainSess.State,
		events:       s.domainEventsToADKEvents(domainSess.Events),
		lastUpdateAt: domainSess.UpdatedAt,
	}
}

// adkToDomainSession converts ADK session to our domain session
func (s *SessionService) adkToDomainSession(adkSess session.Session) (*domainsession.Session, error) {
	if adkSess == nil {
		return nil, errors.ErrInvalidInput
	}

	domainSess := &domainsession.Session{
		AppName:   adkSess.AppName(),
		UserID:    adkSess.UserID(),
		SessionID: adkSess.ID(),
		State:     make(map[string]interface{}),
		UpdatedAt: adkSess.LastUpdateTime(),
	}

	// Copy state
	for key, val := range adkSess.State().All() {
		domainSess.State[key] = val
	}

	return domainSess, nil
}

// adkToDomainEvent converts ADK event to our domain event
func (s *SessionService) adkToDomainEvent(adkEvent *session.Event) (*domainsession.Event, error) {
	if adkEvent == nil {
		return nil, errors.ErrInvalidInput
	}

	// Convert content to map
	contentMap := make(map[string]interface{})
	if adkEvent.LLMResponse.Content != nil {
		// Serialize content to JSON for storage
		contentBytes, err := json.Marshal(adkEvent.LLMResponse.Content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal content")
		}
		if err := json.Unmarshal(contentBytes, &contentMap); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal content")
		}
	}

	// Convert actions
	domainActions := domainsession.EventActions{
		TransferToAgent:   adkEvent.Actions.TransferToAgent,
		Escalate:          adkEvent.Actions.Escalate,
		SkipSummarization: adkEvent.Actions.SkipSummarization,
		StateDelta:        adkEvent.Actions.StateDelta,
	}

	// Convert usage metadata
	var domainUsage *domainsession.UsageMetadata
	if adkEvent.UsageMetadata != nil {
		domainUsage = &domainsession.UsageMetadata{
			PromptTokenCount:     adkEvent.UsageMetadata.PromptTokenCount,
			CandidatesTokenCount: adkEvent.UsageMetadata.CandidatesTokenCount,
			TotalTokenCount:      adkEvent.UsageMetadata.TotalTokenCount,
		}
	}

	return &domainsession.Event{
		EventID:       adkEvent.ID,
		Author:        adkEvent.Author,
		Content:       contentMap,
		Timestamp:     adkEvent.Timestamp,
		Branch:        adkEvent.Branch,
		Partial:       adkEvent.LLMResponse.Partial,
		TurnComplete:  adkEvent.TurnComplete,
		Actions:       domainActions,
		UsageMetadata: domainUsage,
	}, nil
}

// domainEventsToADKEvents converts our domain events to ADK events
func (s *SessionService) domainEventsToADKEvents(domainEvents []domainsession.Event) []*session.Event {
	adkEvents := make([]*session.Event, len(domainEvents))
	for i, domainEvent := range domainEvents {
		adkEvents[i] = s.domainEventToADKEvent(&domainEvent)
	}
	return adkEvents
}

// domainEventToADKEvent converts single domain event to ADK event
func (s *SessionService) domainEventToADKEvent(domainEvent *domainsession.Event) *session.Event {
	// Convert content map to genai.Content
	var content *genai.Content
	if len(domainEvent.Content) > 0 {
		// Deserialize stored content back to genai.Content
		contentBytes, _ := json.Marshal(domainEvent.Content)
		content = &genai.Content{}
		_ = json.Unmarshal(contentBytes, content) // Ignore unmarshal errors for backward compatibility
	}

	// Convert usage metadata
	var usageMeta *genai.GenerateContentResponseUsageMetadata
	if domainEvent.UsageMetadata != nil {
		usageMeta = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     domainEvent.UsageMetadata.PromptTokenCount,
			CandidatesTokenCount: domainEvent.UsageMetadata.CandidatesTokenCount,
			TotalTokenCount:      domainEvent.UsageMetadata.TotalTokenCount,
		}
	}

	// Create event with embedded LLMResponse
	event := &session.Event{
		ID:        domainEvent.EventID,
		Author:    domainEvent.Author,
		Timestamp: domainEvent.Timestamp,
		Branch:    domainEvent.Branch,
		Actions: session.EventActions{
			TransferToAgent:   domainEvent.Actions.TransferToAgent,
			Escalate:          domainEvent.Actions.Escalate,
			SkipSummarization: domainEvent.Actions.SkipSummarization,
			StateDelta:        domainEvent.Actions.StateDelta,
		},
	}

	// Set LLMResponse fields (embedded)
	event.LLMResponse.Content = content
	event.LLMResponse.Partial = domainEvent.Partial
	event.LLMResponse.UsageMetadata = usageMeta

	return event
}

// adkSession is a wrapper that implements session.Session interface
type adkSession struct {
	appName      string
	userID       string
	sessionID    string
	state        map[string]interface{}
	events       []*session.Event
	lastUpdateAt time.Time
}

func (s *adkSession) AppName() string {
	return s.appName
}

func (s *adkSession) UserID() string {
	return s.userID
}

func (s *adkSession) ID() string {
	return s.sessionID
}

func (s *adkSession) State() session.State {
	return &adkState{state: s.state}
}

func (s *adkSession) Events() session.Events {
	return &adkEvents{events: s.events}
}

func (s *adkSession) LastUpdateTime() time.Time {
	return s.lastUpdateAt
}

// adkState implements session.State
type adkState struct {
	state map[string]interface{}
}

func (s *adkState) Get(key string) (interface{}, error) {
	if val, ok := s.state[key]; ok {
		return val, nil
	}
	return nil, session.ErrStateKeyNotExist
}

func (s *adkState) Set(key string, val interface{}) error {
	s.state[key] = val
	return nil
}

func (s *adkState) All() iter.Seq2[string, interface{}] {
	return func(yield func(string, interface{}) bool) {
		for key, val := range s.state {
			if !yield(key, val) {
				return
			}
		}
	}
}

// adkEvents implements session.Events
type adkEvents struct {
	events []*session.Event
}

func (e *adkEvents) Len() int {
	return len(e.events)
}

func (e *adkEvents) At(i int) *session.Event {
	if i < 0 || i >= len(e.events) {
		return nil
	}
	return e.events[i]
}

func (e *adkEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, event := range e.events {
			if !yield(event) {
				return
			}
		}
	}
}
