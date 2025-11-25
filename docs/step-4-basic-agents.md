# Step 4: Basic Agent System

## Overview
Создание базовой агентной системы с Google ADK, AI providers, template system и первыми агентами.

**Цель**: Рабочая система из 3-4 базовых агентов с Chain-of-Thought reasoning.

**Время**: 7-10 дней

---

## 4.1 AI Provider Registry

### Provider Interface
**File**: `internal/adapters/ai/interface.go`

```go
package ai

import (
    "context"
    "google.golang.org/adk/model"
)

type Provider interface {
    Name() string
    GetModel(modelID string) (model.Model, error)
    ListModels() []ModelInfo
    SupportsStreaming() bool
    SupportsTools() bool
}

type ModelInfo struct {
    ID          string
    Name        string
    MaxTokens   int
    InputCost   float64
    OutputCost  float64
}

type ProviderRegistry interface {
    Register(provider Provider)
    Get(name string) (Provider, bool)
    GetModel(providerName, modelID string) (model.Model, error)
    ListProviders() []string
}
```

### Claude Provider
**File**: `internal/adapters/ai/claude/provider.go`

```go
package claude

import (
    "google.golang.org/adk/model"
    "google.golang.org/genai"
    "prometheus/internal/adapters/ai"
)

type Provider struct {
    apiKey string
    models map[string]*modelWrapper
}

func New(apiKey string) *Provider {
    return &Provider{
        apiKey: apiKey,
        models: make(map[string]*modelWrapper),
    }
}

func (p *Provider) Name() string {
    return "claude"
}

func (p *Provider) GetModel(modelID string) (model.Model, error) {
    if m, ok := p.models[modelID]; ok {
        return m, nil
    }

    client := genai.NewClient(&genai.ClientConfig{
        Backend:  genai.BackendAnthropic,
        APIKey:   p.apiKey,
        Project:  "",
    })

    m := &modelWrapper{
        model: client.GenerativeModel(modelID),
    }

    p.models[modelID] = m
    return m, nil
}

func (p *Provider) ListModels() []ai.ModelInfo {
    return []ai.ModelInfo{
        {ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", MaxTokens: 64000},
        {ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", MaxTokens: 200000},
    }
}

func (p *Provider) SupportsStreaming() bool { return true }
func (p *Provider) SupportsTools() bool     { return true }
```

### Registry Implementation
**File**: `internal/adapters/ai/registry.go`

```go
package ai

import (
    "fmt"
    "sync"
    "google.golang.org/adk/model"
)

type registry struct {
    providers map[string]Provider
    mu        sync.RWMutex
}

func NewRegistry() ProviderRegistry {
    return &registry{
        providers: make(map[string]Provider),
    }
}

func (r *registry) Register(provider Provider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[provider.Name()] = provider
}

func (r *registry) Get(name string) (Provider, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    p, ok := r.providers[name]
    return p, ok
}

func (r *registry) GetModel(providerName, modelID string) (model.Model, error) {
    provider, ok := r.Get(providerName)
    if !ok {
        return nil, fmt.Errorf("provider not found: %s", providerName)
    }
    return provider.GetModel(modelID)
}

func (r *registry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}
```

---

## 4.2 Template System

### Template Registry
**File**: `pkg/templates/loader.go`

```go
package templates

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "text/template"
)

type Registry struct {
    templates map[string]*template.Template
    mu        sync.RWMutex
    basePath  string
}

var globalRegistry *Registry

func Init(basePath string) {
    registry, err := loadTemplates(basePath)
    if err != nil {
        panic(fmt.Sprintf("failed to load templates: %v", err))
    }
    globalRegistry = registry
}

func Get() *Registry {
    if globalRegistry == nil {
        panic("templates not initialized")
    }
    return globalRegistry
}

func loadTemplates(basePath string) (*Registry, error) {
    registry := &Registry{
        templates: make(map[string]*template.Template),
        basePath:  basePath,
    }

    err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() || !strings.HasSuffix(path, ".tmpl") {
            return err
        }

        relPath, _ := filepath.Rel(basePath, path)
        id := strings.TrimSuffix(relPath, ".tmpl")
        id = strings.ReplaceAll(id, string(os.PathSeparator), "/")

        content, _ := os.ReadFile(path)
        tmpl, _ := template.New(id).Parse(string(content))

        registry.templates[id] = tmpl
        return nil
    })

    return registry, err
}

func (r *Registry) Lookup(id string) *Template {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tmpl, ok := r.templates[id]
    if !ok {
        return nil
    }
    return &Template{tmpl: tmpl, id: id}
}

type Template struct {
    tmpl *template.Template
    id   string
}

func (t *Template) Render(data interface{}) (string, error) {
    var buf bytes.Buffer
    if err := t.tmpl.Execute(&buf, data); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

### Example Prompt Template
**File**: `templates/agents/market_analyst/system.tmpl`

```
You are an expert cryptocurrency market analyst with Chain-of-Thought reasoning capabilities.

## Your Mission
Analyze {{.Symbol}} market using available tools.

## Available Tools
{{range .Tools}}
- **{{.Name}}**: {{.Description}}
{{end}}

## Chain-of-Thought Process

**STEP 1 - DATA GATHERING:**
<thinking>
Decide which tools to use and why.
</thinking>

**STEP 2 - ANALYSIS:**
<thinking>
Analyze tool results. What patterns do you see?
</thinking>

**STEP 3 - SYNTHESIS:**
<thinking>
Combine all data into actionable insight.
</thinking>

## Output Format
Return JSON with:
{
  "signal": "buy|sell|hold",
  "confidence": 75,
  "reasoning": "Complete reasoning chain",
  "key_levels": {"support": [], "resistance": []},
  "tool_calls_made": 7
}

## Risk Profile: {{.RiskLevel}}
{{if eq .RiskLevel "conservative"}}
- Require high confidence (>75%)
- Prefer trend-following
{{else if eq .RiskLevel "aggressive"}}
- Accept moderate confidence (>60%)
- Consider counter-trend
{{end}}
```

---

## 4.3 Tools Registry

### Tool Registry
**File**: `internal/tools/registry.go`

```go
package tools

import (
    "sync"
    "google.golang.org/adk/tool"
)

type Registry struct {
    tools map[string]tool.Tool
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]tool.Tool),
    }
}

func (r *Registry) Register(name string, t tool.Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[name] = t
}

func (r *Registry) Get(name string) (tool.Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    t, ok := r.tools[name]
    return t, ok
}

func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    return names
}
```

### Example Tool: Get Price
**File**: `internal/tools/market/get_price.go`

```go
package market

import (
    "context"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"
    "prometheus/internal/adapters/exchanges"
)

type GetPriceArgs struct {
    Symbol string `json:"symbol" jsonschema:"description=Trading pair (e.g. BTCUSDT),required"`
}

type GetPriceResult struct {
    Symbol    string  `json:"symbol"`
    Price     float64 `json:"price"`
    Change24h float64 `json:"change_24h"`
    Volume24h float64 `json:"volume_24h"`
}

type GetPriceTool struct {
    factory exchanges.CentralFactory
}

func NewGetPriceTool(factory exchanges.CentralFactory) tool.Tool {
    t := &GetPriceTool{factory: factory}
    return functiontool.New(
        "get_price",
        "Get current price and 24h stats for a trading pair",
        t.Execute,
    )
}

func (t *GetPriceTool) Execute(ctx tool.Context, args GetPriceArgs) (GetPriceResult, error) {
    client, err := t.factory.GetClient("binance")
    if err != nil {
        return GetPriceResult{}, err
    }

    ticker, err := client.GetTicker(context.Background(), args.Symbol)
    if err != nil {
        return GetPriceResult{}, err
    }

    return GetPriceResult{
        Symbol:    ticker.Symbol,
        Price:     ticker.Price.InexactFloat64(),
        Change24h: ticker.Change24h.InexactFloat64(),
        Volume24h: ticker.Volume24h.InexactFloat64(),
    }, nil
}
```

### Example Tool: RSI
**File**: `internal/tools/indicators/rsi.go`

```go
package indicators

import (
    "context"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"
    "prometheus/internal/repository/clickhouse"
)

type RSIArgs struct {
    Symbol    string `json:"symbol" jsonschema:"description=Trading pair,required"`
    Timeframe string `json:"timeframe" jsonschema:"description=Timeframe (1h/4h/1d),required"`
    Period    int    `json:"period" jsonschema:"description=RSI period (default 14)"`
}

type RSIResult struct {
    RSI     float64 `json:"rsi"`
    Signal  string  `json:"signal"`  // overbought/oversold/neutral
    PrevRSI float64 `json:"prev_rsi"`
}

type RSITool struct {
    repo *clickhouse.MarketDataRepository
}

func NewRSITool(repo *clickhouse.MarketDataRepository) tool.Tool {
    t := &RSITool{repo: repo}
    return functiontool.New(
        "rsi",
        "Calculate Relative Strength Index. RSI > 70 = overbought, RSI < 30 = oversold",
        t.Calculate,
    )
}

func (t *RSITool) Calculate(ctx tool.Context, args RSIArgs) (RSIResult, error) {
    period := args.Period
    if period == 0 {
        period = 14
    }

    candles, err := t.repo.GetLatestOHLCV(context.Background(), "binance", args.Symbol, args.Timeframe, period+50)
    if err != nil {
        return RSIResult{}, err
    }

    closes := make([]float64, len(candles))
    for i, c := range candles {
        closes[i] = c.Close.InexactFloat64()
    }

    rsi := calculateRSI(closes, period)
    
    signal := "neutral"
    if rsi > 70 {
        signal = "overbought"
    } else if rsi < 30 {
        signal = "oversold"
    }

    return RSIResult{
        RSI:    rsi,
        Signal: signal,
    }, nil
}

func calculateRSI(closes []float64, period int) float64 {
    if len(closes) < period+1 {
        return 50
    }

    gains, losses := 0.0, 0.0
    for i := 1; i <= period; i++ {
        change := closes[len(closes)-period-1+i] - closes[len(closes)-period-2+i]
        if change > 0 {
            gains += change
        } else {
            losses -= change
        }
    }

    avgGain := gains / float64(period)
    avgLoss := losses / float64(period)

    if avgLoss == 0 {
        return 100
    }

    rs := avgGain / avgLoss
    return 100 - (100 / (1 + rs))
}
```

---

## 4.4 Agent Factory

### Factory
**File**: `internal/agents/factory.go`

```go
package agents

import (
    "fmt"
    "google.golang.org/adk/agent"
    "google.golang.org/adk/agent/llmagent"
    "google.golang.org/adk/tool"
    "prometheus/internal/adapters/ai"
    "prometheus/internal/tools"
    "prometheus/pkg/templates"
)

type Factory struct {
    aiRegistry   ai.ProviderRegistry
    toolRegistry *tools.Registry
    templates    *templates.Registry
}

type FactoryDeps struct {
    AIRegistry   ai.ProviderRegistry
    ToolRegistry *tools.Registry
    Templates    *templates.Registry
}

func NewFactory(deps FactoryDeps) *Factory {
    return &Factory{
        aiRegistry:   deps.AIRegistry,
        toolRegistry: deps.ToolRegistry,
        templates:    deps.Templates,
    }
}

type AgentConfig struct {
    Type                 AgentType
    Name                 string
    AIProvider           string
    Model                string
    Tools                []string
    SystemPromptTemplate string
    MaxToolCalls         int
}

func (f *Factory) CreateAgent(cfg AgentConfig) (agent.Agent, error) {
    // Get AI model
    model, err := f.aiRegistry.GetModel(cfg.AIProvider, cfg.Model)
    if err != nil {
        return nil, fmt.Errorf("failed to get model: %w", err)
    }

    // Get tools
    agentTools := make([]tool.Tool, 0, len(cfg.Tools))
    for _, toolName := range cfg.Tools {
        t, ok := f.toolRegistry.Get(toolName)
        if !ok {
            return nil, fmt.Errorf("tool not found: %s", toolName)
        }
        agentTools = append(agentTools, t)
    }

    // Render system prompt
    systemPrompt := ""
    if cfg.SystemPromptTemplate != "" {
        tmpl := f.templates.Lookup(cfg.SystemPromptTemplate)
        if tmpl != nil {
            systemPrompt, _ = tmpl.Render(map[string]interface{}{
                "Symbol":    "BTC/USDT",
                "RiskLevel": "conservative",
                "Tools":     agentTools,
            })
        }
    }

    // Create LLM agent
    return llmagent.New(llmagent.Config{
        Name:        cfg.Name,
        Description: fmt.Sprintf("%s agent", cfg.Name),
        Model:       model,
        Tools:       agentTools,
        Instruction: systemPrompt,
    })
}

type AgentType string

const (
    AgentMarketAnalyst AgentType = "market_analyst"
    AgentRiskManager   AgentType = "risk_manager"
    AgentExecutor      AgentType = "executor"
)
```

---

## 4.5 Tool Assignments

### Agent Tool Map
**File**: `internal/agents/tool_assignments.go`

```go
package agents

var AgentToolMap = map[AgentType][]string{
    AgentMarketAnalyst: {
        "get_price",
        "get_ohlcv",
        "get_orderbook",
        "rsi",
        "macd",
        "bollinger",
        "atr",
        "ema",
        "vwap",
        "volume_profile",
    },

    AgentRiskManager: {
        "get_positions",
        "get_balance",
        "get_daily_pnl",
        "calculate_position_size",
        "validate_trade",
        "check_circuit_breaker",
    },

    AgentExecutor: {
        "get_balance",
        "place_order",
        "cancel_order",
        "set_leverage",
    },
}
```

---

## 4.6 Update Main

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "prometheus/internal/adapters/ai"
    "prometheus/internal/adapters/ai/claude"
    "prometheus/internal/adapters/config"
    "prometheus/internal/adapters/exchanges"
    "prometheus/internal/agents"
    "prometheus/internal/tools"
    marketTools "prometheus/internal/tools/market"
    "prometheus/pkg/logger"
    "prometheus/pkg/templates"
)

func main() {
    cfg, _ := config.Load()
    logger.Init(cfg.App.LogLevel, cfg.App.Env)
    defer logger.Sync()

    log := logger.Get()
    log.Info("Starting Prometheus Trading System")

    // Initialize templates FIRST
    templates.Init("templates")
    log.Infof("✓ Loaded %d templates", len(templates.Get().List()))

    // Initialize AI providers
    aiRegistry := ai.NewRegistry()
    if cfg.AI.ClaudeKey != "" {
        aiRegistry.Register(claude.New(cfg.AI.ClaudeKey))
    }
    log.Infof("✓ Registered AI providers: %v", aiRegistry.ListProviders())

    // Initialize exchange factory
    centralFactory := exchanges.NewCentralFactory(cfg.MarketData)

    // Initialize tools registry
    toolRegistry := tools.NewRegistry()
    toolRegistry.Register("get_price", marketTools.NewGetPriceTool(centralFactory))
    // ... register more tools
    log.Infof("✓ Registered %d tools", len(toolRegistry.List()))

    // Initialize agent factory
    agentFactory := agents.NewFactory(agents.FactoryDeps{
        AIRegistry:   aiRegistry,
        ToolRegistry: toolRegistry,
        Templates:    templates.Get(),
    })

    // Create Market Analyst agent
    marketAnalyst, _ := agentFactory.CreateAgent(agents.AgentConfig{
        Type:                 agents.AgentMarketAnalyst,
        Name:                 "MarketAnalyst",
        AIProvider:           "claude",
        Model:                "claude-3-5-sonnet-20241022",
        Tools:                agents.AgentToolMap[agents.AgentMarketAnalyst],
        SystemPromptTemplate: "agents/market_analyst/system",
        MaxToolCalls:         25,
    })

    log.Info("✓ Agents initialized")
    log.Info("System ready")

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Info("Shutting down...")
}
```

---

## Checklist

- [ ] AI provider registry implemented
- [ ] Claude provider working
- [ ] Template system loaded
- [ ] Tools registry created
- [ ] Basic tools implemented (get_price, rsi)
- [ ] Agent factory created
- [ ] First agent (Market Analyst) created
- [ ] Agent can call tools
- [ ] Chain-of-Thought working

---

## Next Step

**→ [Step 5: Advanced Agents & Tools](step-5-advanced-agents.md)**

Добавим:
- Все остальные агенты (SMC, Sentiment, Strategy Planner)
- Полный набор tools (80+)
- Sequential/Parallel agent workflows
- Memory system

