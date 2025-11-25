# Prometheus - Agentic Trading System

Autonomous multi-agent trading system for cryptocurrency markets with Telegram management.

## Features

- **Multi-Agent Architecture**: Specialized agents for analysis, risk management, and execution
- **Multi-Provider AI**: Claude, OpenAI, DeepSeek, Gemini via unified interface
- **Multi-Exchange**: Binance, Bybit, OKX support
- **Real-time Data Pipeline**: Market data, sentiment, news, macro, derivatives
- **Telegram-first UX**: Complete system management via Telegram bot
- **Risk Engine**: Independent service with circuit breaker and kill switch
- **Self-Evaluation**: Automatic strategy performance analysis

## Tech Stack

- **Language**: Go 1.22+
- **Agent Framework**: Google ADK Go
- **Databases**: PostgreSQL 16 + pgvector, ClickHouse
- **Cache**: Redis 7
- **Message Queue**: Apache Kafka
- **Logging**: Zap (structured logging)

## Project Structure

```
prometheus/
├── cmd/                    # Application entrypoint
├── internal/              # Internal packages
│   ├── adapters/         # External adapters (DB, APIs, etc.)
│   ├── agents/           # AI agents
│   ├── domain/           # Business domain entities
│   ├── tools/            # Agent tools
│   ├── workers/          # Background workers
│   └── repository/       # Data repositories
├── pkg/                  # Public packages
│   ├── logger/          # Logging utilities
│   ├── templates/       # Prompt templates
│   └── errors/          # Error tracking
├── migrations/          # Database migrations
└── docs/               # Documentation
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16
- ClickHouse
- Redis
- Kafka

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd prometheus
```

2. Copy environment configuration:
```bash
cp .env.local.example .env
# Edit .env with your configuration
```

See [docs/ENV_SETUP.md](docs/ENV_SETUP.md) for detailed environment setup.

3. Install dependencies:
```bash
go mod download
```

4. Start infrastructure with Docker Compose:
```bash
docker-compose up -d
```

5. Run database migrations:
```bash
make migrate-up
```

6. Run the application:
```bash
go run cmd/main.go
```

## Development

### Project Setup (Completed ✅)
- [x] Initialize Go project structure
- [x] Setup go.mod with dependencies
- [x] Configure environment management
- [x] Setup environment documentation
- [x] Initialize structured logging with error tracking

### Next Steps
See [docs/DEVELOPMENT_PLAN.md](docs/DEVELOPMENT_PLAN.md) for the complete development roadmap.

## Documentation

- [Technical Specification](docs/specs.md)
- [Development Plan](docs/DEVELOPMENT_PLAN.md)
- [Environment Setup](docs/ENV_SETUP.md)

## Architecture

The system uses a multi-agent architecture where specialized AI agents handle different aspects of trading:

1. **Analysis Agents**: Market, Sentiment, On-Chain, SMC, Order Flow, Derivatives
2. **Decision Agents**: Strategy Planner, Risk Manager
3. **Execution Agents**: Executor, Position Manager
4. **Evaluation Agents**: Self Evaluator

All agents communicate via Kafka and share memory via PostgreSQL with pgvector.

## License

Proprietary - All rights reserved

## Contact

For questions and support, please open an issue.


Autonomous multi-agent trading system for cryptocurrency markets with Telegram management.

## Features

- **Multi-Agent Architecture**: Specialized agents for analysis, risk management, and execution
- **Multi-Provider AI**: Claude, OpenAI, DeepSeek, Gemini via unified interface
- **Multi-Exchange**: Binance, Bybit, OKX support
- **Real-time Data Pipeline**: Market data, sentiment, news, macro, derivatives
- **Telegram-first UX**: Complete system management via Telegram bot
- **Risk Engine**: Independent service with circuit breaker and kill switch
- **Self-Evaluation**: Automatic strategy performance analysis

## Tech Stack

- **Language**: Go 1.22+
- **Agent Framework**: Google ADK Go
- **Databases**: PostgreSQL 16 + pgvector, ClickHouse
- **Cache**: Redis 7
- **Message Queue**: Apache Kafka
- **Logging**: Zap (structured logging)

## Project Structure

```
prometheus/
├── cmd/                    # Application entrypoint
├── internal/              # Internal packages
│   ├── adapters/         # External adapters (DB, APIs, etc.)
│   ├── agents/           # AI agents
│   ├── domain/           # Business domain entities
│   ├── tools/            # Agent tools
│   ├── workers/          # Background workers
│   └── repository/       # Data repositories
├── pkg/                  # Public packages
│   ├── logger/          # Logging utilities
│   ├── templates/       # Prompt templates
│   └── errors/          # Error tracking
├── migrations/          # Database migrations
└── docs/               # Documentation
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16
- ClickHouse
- Redis
- Kafka

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd prometheus
```

2. Copy environment configuration:
```bash
cp .env.local.example .env
# Edit .env with your configuration
```

See [docs/ENV_SETUP.md](docs/ENV_SETUP.md) for detailed environment setup.

3. Install dependencies:
```bash
go mod download
```

4. Start infrastructure with Docker Compose:
```bash
docker-compose up -d
```

5. Run database migrations:
```bash
make migrate-up
```

6. Run the application:
```bash
go run cmd/main.go
```

## Development

### Project Setup (Completed ✅)
- [x] Initialize Go project structure
- [x] Setup go.mod with dependencies
- [x] Configure environment management
- [x] Setup environment documentation
- [x] Initialize structured logging with error tracking

### Next Steps
See [docs/DEVELOPMENT_PLAN.md](docs/DEVELOPMENT_PLAN.md) for the complete development roadmap.

## Documentation

- [Technical Specification](docs/specs.md)
- [Development Plan](docs/DEVELOPMENT_PLAN.md)
- [Environment Setup](docs/ENV_SETUP.md)

## Architecture

The system uses a multi-agent architecture where specialized AI agents handle different aspects of trading:

1. **Analysis Agents**: Market, Sentiment, On-Chain, SMC, Order Flow, Derivatives
2. **Decision Agents**: Strategy Planner, Risk Manager
3. **Execution Agents**: Executor, Position Manager
4. **Evaluation Agents**: Self Evaluator

All agents communicate via Kafka and share memory via PostgreSQL with pgvector.

## License

Proprietary - All rights reserved

## Contact

For questions and support, please open an issue.

