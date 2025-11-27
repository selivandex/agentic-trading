<!-- @format -->

# Prometheus Trading System — Complete Flow

**Version:** 1.0  
**Date:** November 27, 2025  
**Purpose:** High-level system flow description without technical implementation details

---

## Overview

Prometheus is an autonomous multi-agent trading system that continuously monitors crypto markets, identifies trading opportunities, and executes trades on behalf of multiple users. The system operates in several independent but coordinated workflows.

---

## Core Principles

1. **Separation of concerns**: Market analysis is performed once globally, personal decisions are made per user
2. **Fail-safe execution**: Multiple validation layers before any real trade
3. **Explainable decisions**: Every decision is logged with structured reasoning
4. **Multi-user isolation**: Each user has separate capital, risk limits, and preferences

---

## 1. Market Research Phase (Global, Runs Once Per Symbol)

### 1.1 Objective

Analyze the market comprehensively to find high-quality trading opportunities that could be valuable for any user.

### 1.2 Flow

```
Market Research Worker (runs every 1-5 minutes)
    ↓
[Step 1] Parallel Analysis (8 specialist agents run simultaneously)
    ├─ Market Analyst: Technical indicators, chart patterns, support/resistance
    ├─ SMC Analyst: Smart Money Concepts (liquidity zones, order blocks, fair value gaps)
    ├─ Sentiment Analyst: News, social media sentiment, fear & greed index
    ├─ Order Flow Analyst: Tape reading, volume delta, large trades detection
    ├─ Derivatives Analyst: Options flow, gamma exposure, max pain analysis
    ├─ Macro Analyst: Economic calendar, Fed decisions, inflation data
    ├─ On-Chain Analyst: Whale movements, exchange flows, MVRV ratio
    └─ Correlation Analyst: BTC vs stocks, dollar index, gold correlation
    ↓
[Step 2] Opportunity Synthesizer (makes decision)
    • Counts consensus: How many analysts agree on direction?
    • Calculates weighted confidence: Different analysts have different weights
    • Identifies conflicts: Does macro contradict technical?
    • Resolves conflicts: Uses tie-breaking rules
    • Makes decision: Publish signal OR skip
    ↓
[Decision Point]
    IF confidence >= 65% AND consensus >= 5/8 analysts AND no major conflicts
        → Publish opportunity signal to queue
    ELSE
        → Skip (document why in logs)
```

### 1.3 Output

**Opportunity Signal** (if published):
- Direction: Long or Short
- Entry price range
- Stop-loss level
- Take-profit targets (multiple levels)
- Risk/Reward ratio
- Confidence score
- Time validity (signal expires in 15-30 minutes)
- Supporting analysis summary from all 8 analysts

---

## 2. Opportunity Queue (Holding Area)

### 2.1 Purpose

Stores published opportunities with expiration time, allowing each user to process them independently.

### 2.2 Flow

```
Opportunity Signal
    ↓
Stored in queue with:
    • Unique ID
    • Symbol (e.g., BTC/USDT)
    • Direction, entry, SL, TP
    • Confidence score
    • Created timestamp
    • Expires at (timestamp + TTL)
    • Status: PENDING
    ↓
Broadcast to all active users
    ↓
Each user's Personal Trading Workflow picks it up
```

### 2.3 Signal Lifecycle

- **PENDING**: Just published, waiting for users to process
- **ACCEPTED**: User's workflow approved and executed the trade
- **REJECTED**: User's risk manager rejected the trade
- **EXPIRED**: Signal TTL expired, no longer valid

---

## 3. Personal Trading Phase (Per User, Runs for Each Active User)

### 3.1 Objective

Receive a pre-analyzed opportunity signal and decide whether to execute it based on **personal context**: available capital, current positions, risk tolerance, trading history.

### 3.2 Flow

```
User receives Opportunity Signal
    ↓
[Step 1] Strategy Planner (personalization)
    • Reads opportunity signal (global market analysis)
    • Reads user context:
        - Available balance
        - Current open positions
        - Risk tolerance (conservative/moderate/aggressive)
        - Preferred leverage
        - Past trading performance
    • Creates personal trading plan:
        - Position size (based on capital and risk)
        - Adjusted entry/SL/TP (if needed)
        - Margin mode (cross/isolated)
        - Leverage (if futures)
    ↓
[Step 2] Risk Manager (validation)
    • Validates position size:
        - Not too large (max % of portfolio)
        - Respects user's risk limits
    • Checks risk engine:
        - Is circuit breaker active? (too many losses today)
        - Is max drawdown exceeded?
        - Is max exposure limit reached?
    • Validates R:R ratio (must be >= 2:1)
    • Validates margin requirements (futures)
    ↓
[Decision Point]
    IF all checks pass
        → Approve plan, send to Executor
    ELSE
        → Reject with reason OR adjust plan (reduce size) and re-validate
    ↓
[Step 3] Executor (order placement)
    • Places order on exchange (uses user's API keys)
    • Handles order types:
        - Simple: Market/Limit order with SL/TP
        - Bracket: Entry + Stop-loss + Take-profit in one
        - Ladder: Entry + Stop-loss + Multiple take-profits
    • Confirms order execution
    • Saves position to database
    ↓
Position is now ACTIVE
```

### 3.3 Output

**Active Position** record:
- User ID
- Symbol
- Direction (Long/Short)
- Entry price (actual filled price)
- Position size
- Stop-loss level
- Take-profit levels
- Current PnL
- Status: OPEN

---

## 4. Position Monitoring Phase (Continuous, Per Active Position)

### 4.1 Objective

Monitor all active positions in real-time and make adjustments when needed: move stop-loss to breakeven, close early if market conditions change, or extend take-profit if momentum is strong.

### 4.2 Flow

```
Position Monitor Worker (runs every 30-60 seconds)
    ↓
For each active position:
    ↓
[Step 1] Position Manager (analysis)
    • Fetches current price
    • Calculates current PnL
    • Checks if any conditions met:
        - Price moved X% in profit → move SL to breakeven?
        - Market structure broken → close early?
        - Strong momentum → trail stop-loss?
        - Partial TP hit → close partial position?
    • Checks risk events:
        - Circuit breaker triggered → close all positions?
        - Max drawdown exceeded → emergency exit?
    • Makes recommendation:
        - HOLD: Do nothing
        - MODIFY: Adjust SL/TP
        - CLOSE_PARTIAL: Take partial profit
        - CLOSE_ALL: Exit position
    ↓
[Step 2] Risk Manager (re-validation)
    • Validates recommendation
    • Checks if action is safe
    • Approves or rejects
    ↓
[Step 3] Executor (modification)
    • Executes approved action:
        - Modify SL/TP on exchange
        - Close position (market order)
        - Cancel existing orders
    ↓
Position updated in database
```

### 4.3 Automatic Position Closure

Position closes automatically when:
- **Stop-loss hit**: Price reached SL level → close immediately to prevent further loss
- **Take-profit hit**: Price reached TP level → close position with profit
- **Circuit breaker**: User hit max drawdown or max consecutive losses → emergency close all
- **Expiration**: User set time-based exit (e.g., close after 24 hours)
- **Margin call**: Insufficient margin (futures) → forced liquidation by exchange

---

## 5. Post-Trade Evaluation Phase (After Each Position Close)

### 5.1 Objective

Analyze the closed trade to understand what worked, what didn't, and extract lessons for future improvements.

### 5.2 Flow

```
Position closed (SL/TP hit or manual close)
    ↓
[Triggered] Self Evaluator Agent
    • Retrieves trade history:
        - Entry/exit prices
        - Duration
        - PnL
        - Original opportunity signal (which analysts supported it?)
        - Market conditions during trade
    • Analyzes outcome:
        - Was the trade plan good?
        - Was entry timing good?
        - Was SL placement good?
        - Did we exit too early/late?
        - Which analyst's analysis was most accurate?
    • Categorizes trade:
        - Perfect execution + win
        - Good plan but stopped out (acceptable loss)
        - Bad plan → lesson learned
        - Winning trade but could be better
    • Generates lessons:
        - "SMC liquidity zone was accurate signal"
        - "Macro headwinds underestimated"
        - "Entered too early, should wait for confirmation"
    ↓
[Output 1] Journal Entry (saved to database)
    • Trade details
    • Analysis
    • Lessons learned
    • Rating: 1-5 stars
    ↓
[Output 2] Strategy Statistics Update
    • Win rate per strategy type
    • Average R:R per strategy
    • Best performing timeframes
    • Best performing market conditions
    ↓
[Output 3] Lesson Promotion (if validated)
    IF lesson quality high AND confirmed by multiple trades
        → Promote to Collective Memory
        → Available to all users (anonymized)
```

---

## 6. Memory System (Continuous Learning)

### 6.1 Two Types of Memory

#### Personal Memory (Per User)
- User's own observations
- Personal trading history
- Preferred strategies
- Risk tolerance adjustments
- Personalized lessons

#### Collective Memory (Shared Across All Users)
- Validated patterns that work
- High-quality lessons (confirmed by 3+ successful trades)
- Market regime detection
- Common mistakes to avoid

### 6.2 Memory Usage

Agents use memory during analysis:
- **Market Analyst**: "Last time BTC was at this level with RSI divergence, it reversed"
- **Strategy Planner**: "User historically performs better with trend-following strategies"
- **Risk Manager**: "This user tends to exit winners too early, recommend higher TP"

---

## 7. Risk Engine (Continuous Protection)

### 7.1 Circuit Breaker Rules

The risk engine stops trading automatically when:

| Condition | Action |
|-----------|--------|
| Daily loss >= X% of account | Pause trading until next day |
| Consecutive losses >= N trades | Pause for manual review |
| Open positions exposure >= Y% | Block new positions |
| Exchange API errors | Pause trading, send alert |
| Unusual market volatility | Reduce position sizes |

### 7.2 Kill Switch

**Emergency shutdown** triggered by:
- User command from Telegram: `/killswitch`
- System detects critical error
- Exchange connection lost
- Suspicious activity detected

**Action**: Close all open positions immediately, cancel all pending orders, pause all trading.

---

## 8. Daily Reporting Phase (Once Per Day)

### 8.1 Objective

Provide users with comprehensive performance summary and improvement recommendations.

### 8.2 Flow

```
Daily Report Worker (runs at 00:00 UTC)
    ↓
For each user:
    ↓
Self Evaluator Agent generates report:
    • Performance summary:
        - Total PnL (today, this week, this month)
        - Win rate
        - Average R:R
        - Best/worst trades
    • Strategy breakdown:
        - Which strategies performed best
        - Which market conditions were favorable
        - Which timeframes worked
    • Improvement recommendations:
        - "Your stop-losses are too tight, consider wider stops"
        - "You perform better in trending markets, avoid choppy conditions"
        - "SMC-based entries have 70% win rate for you, increase allocation"
    • Risk metrics:
        - Max drawdown
        - Sharpe ratio
        - Number of circuit breaker triggers
    ↓
Report sent via Telegram
```

---

## 9. User Interaction (Telegram Bot)

### 9.1 User Can

**Monitor**:
- `/balance` - Check account balance
- `/positions` - View open positions with PnL
- `/stats` - Trading statistics
- `/report` - Generate performance report

**Control**:
- `/pause` - Pause trading (stop processing new opportunities)
- `/resume` - Resume trading
- `/killswitch` - Emergency close all positions
- `/settings` - Adjust risk tolerance, preferred leverage, etc.

**Configure**:
- `/connect bybit API_KEY SECRET` - Connect exchange account
- `/disconnect bybit` - Remove exchange
- `/set-risk moderate` - Change risk profile

**Opportunities**:
- Receive opportunity notifications: "🎯 New signal: BTC/USDT LONG at $43,200"
- Accept/reject manually (optional): "✅ Accept" or "❌ Skip"

---

## 10. Multi-User Execution Model

### 10.1 How Multiple Users Are Handled

```
Market Research (runs ONCE)
    ↓
Opportunity Signal published
    ↓
┌─────────────────┬─────────────────┬─────────────────┐
│   User A        │   User B        │   User C        │
│   (Conservative)│   (Aggressive)  │   (Moderate)    │
├─────────────────┼─────────────────┼─────────────────┤
│ Strategy Planner│ Strategy Planner│ Strategy Planner│
│ → Small size    │ → Large size    │ → Medium size   │
│ → 2x leverage   │ → 10x leverage  │ → 5x leverage   │
├─────────────────┼─────────────────┼─────────────────┤
│ Risk Manager    │ Risk Manager    │ Risk Manager    │
│ → APPROVED      │ → REJECTED      │ → APPROVED      │
│                 │ (too risky)     │                 │
├─────────────────┼─────────────────┼─────────────────┤
│ Executor        │ (skipped)       │ Executor        │
│ ✅ Position OPEN│                 │ ✅ Position OPEN│
└─────────────────┴─────────────────┴─────────────────┘
```

**Key point**: Same opportunity, different personal decisions based on individual risk profiles and capital.

---

## 11. Data Collection (Background Workers)

### 11.1 Continuous Data Pipeline

System collects market data in background to feed analysts:

| Worker | Frequency | Data Type |
|--------|-----------|-----------|
| OHLCV Collector | 1 minute | Candlestick data |
| Ticker Collector | 5 seconds | Real-time prices |
| OrderBook Collector | 10 seconds | Order book depth |
| Trades Collector | 1 second | Tape / recent trades |
| Funding Collector | 1 minute | Futures funding rates |
| Liquidations Collector | 5 seconds | Liquidation events |
| News Collector | 5 minutes | Crypto news |
| Sentiment Collector | 2 minutes | Social media sentiment |
| On-Chain Collector | 5 minutes | Whale movements, flows |
| Macro Collector | 1 hour | Economic calendar |
| Options Collector | 5 minutes | Options flow, gamma |

All data is stored in high-performance databases for fast retrieval during analysis.

---

## 12. System Safeguards

### 12.1 Multiple Protection Layers

1. **Pre-trade validation** (Risk Manager):
   - Position size limits
   - Risk/Reward validation
   - Circuit breaker check
   - Margin requirements

2. **Exchange-level protection**:
   - Stop-loss orders placed immediately
   - Take-profit orders set
   - Leverage limits enforced

3. **Real-time monitoring**:
   - Position monitoring every 30-60s
   - Risk metrics recalculated continuously
   - Circuit breaker can trigger anytime

4. **Emergency controls**:
   - Kill switch (user or system)
   - Manual pause via Telegram
   - Automatic trading pause on errors

---

## 13. Complete User Journey Example

### Example: Alice, Conservative Trader

**9:00 AM** - Market Research runs for BTC/USDT
- 8 analysts analyze the market
- 6/8 analysts signal LONG
- Opportunity Synthesizer: 72% confidence → **PUBLISH**

**9:01 AM** - Alice's Personal Trading Workflow
- **Strategy Planner**: Creates plan for $100 position, 2x leverage, tight SL
- **Risk Manager**: Validates → APPROVED (fits Alice's conservative profile)
- **Executor**: Places bracket order on Bybit
- Position opened: BTC LONG @ $43,200, SL: $42,800, TP: $44,400

**9:01 AM - 11:30 AM** - Position Monitoring
- Price moves to $43,600 (+$80 profit)
- Position Manager: Move SL to breakeven ($43,200)
- Executor: Updates stop-loss on exchange

**11:35 AM** - Take-profit hit
- Price reaches $44,400
- Position auto-closes with +$240 profit (+2.8% ROI)

**11:36 AM** - Post-Trade Evaluation
- Self Evaluator analyzes: "Perfect execution, SMC liquidity grab signal was key"
- Journal entry created
- Lesson learned: "Order flow divergence + SMC signal = high confidence"

**12:00 AM (next day)** - Daily Report
- Alice receives Telegram message:
  - "📊 Daily Report: +$240 (+2.8%), 1 trade, 100% win rate"
  - "💡 Recommendation: Your entries are excellent, consider larger position sizes"

---

## 14. Failure Scenarios & Handling

### 14.1 What If...

**Market Research finds no opportunities?**
- No signal published
- System waits for next analysis cycle
- No unnecessary trades (quality > quantity)

**Risk Manager rejects all users?**
- Signal expires
- No trades executed
- Better safe than sorry

**Position goes against user immediately?**
- Stop-loss triggers automatically
- Loss is contained to predefined amount
- Self Evaluator analyzes if entry was bad or just bad luck

**Exchange API connection fails?**
- Trading pauses automatically
- Alert sent to user
- Existing positions monitored via backup connection
- System waits for connection recovery

**Circuit breaker triggers?**
- All new trading paused
- Open positions stay open (unless kill switch triggered)
- User notified via Telegram
- Manual review required to resume

---

## 15. Success Metrics

### 15.1 System Performance

- **Opportunity quality**: % of published signals that result in profitable trades
- **Win rate**: % of trades that reach TP vs SL
- **Average R:R**: Realized risk/reward ratio across all trades
- **AI cost efficiency**: Cost per published signal
- **User satisfaction**: % of users actively trading after 30 days

### 15.2 Agent Performance

- **Analyst accuracy**: Which analysts' signals correlate with wins
- **Synthesis quality**: OpportunitySynthesizer's consensus accuracy
- **Risk manager effectiveness**: % of avoided bad trades
- **Self-evaluation quality**: Usefulness of generated lessons

---

## End of System Flow

This flow describes the complete lifecycle from market analysis to trade execution, monitoring, evaluation, and continuous improvement. Each phase is independent but coordinated through well-defined data contracts (opportunity signals, position records, journal entries, memory items).

