-- ADK Sessions table
CREATE TABLE
  IF NOT EXISTS adk_sessions (
    id UUID PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    state JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    UNIQUE (app_name, user_id, session_id)
  );

CREATE INDEX IF NOT EXISTS idx_adk_sessions_app_user ON adk_sessions (app_name, user_id);

CREATE INDEX IF NOT EXISTS idx_adk_sessions_updated ON adk_sessions (updated_at DESC);

-- ADK Events table  
CREATE TABLE
  IF NOT EXISTS adk_events (
    id UUID PRIMARY KEY,
    session_uuid UUID NOT NULL REFERENCES adk_sessions (id) ON DELETE CASCADE,
    event_id VARCHAR(255) NOT NULL UNIQUE,
    author VARCHAR(255) NOT NULL,
    content JSONB NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    branch VARCHAR(255),
    partial BOOLEAN NOT NULL DEFAULT FALSE,
    turn_complete BOOLEAN NOT NULL DEFAULT FALSE,
    actions JSONB NOT NULL DEFAULT '{}',
    usage_metadata JSONB
  );

CREATE INDEX IF NOT EXISTS idx_adk_events_session ON adk_events (session_uuid, timestamp);

CREATE INDEX IF NOT EXISTS idx_adk_events_author ON adk_events (author);

CREATE INDEX IF NOT EXISTS idx_adk_events_timestamp ON adk_events (timestamp DESC);

-- ADK App State table (application-level state shared across all users)
CREATE TABLE
  IF NOT EXISTS adk_app_state (
    app_name VARCHAR(255) PRIMARY KEY,
    state JSONB NOT NULL DEFAULT '{}'
  );

-- ADK User State table (user-level state shared across user's sessions)
CREATE TABLE
  IF NOT EXISTS adk_user_state (
    app_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    state JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (app_name, user_id)
  );

CREATE INDEX IF NOT EXISTS idx_adk_user_state_app ON adk_user_state (app_name);
