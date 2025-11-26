# Agent Prompt Templates - CoT Rewrite Guide

## Overview

This guide explains how to rewrite agent prompts to support **real Chain-of-Thought (CoT) reasoning** with Google ADK multi-turn conversation capabilities.

---

## Problem with Current Prompts

**Current approach (WRONG):**
- Prompts end with large JSON structures
- Agents are instructed to return `{"analysis": {...}, "setups": [...]}`
- This is **one-shot reasoning** - agent thinks internally and returns a static blob
- No real Chain-of-Thought, no iterative tool usage

**Example of what NOT to do:**
```
# FINAL OUTPUT

Return this JSON:
{
  "analysis_summary": "...",
  "bias": {...},
  "setups": [...]
}
```

---

## New CoT Approach (CORRECT)

**What we want:**
- Agent thinks aloud, step-by-step
- Agent calls tools **iteratively** (one at a time)
- Agent reflects on each tool result
- Agent saves conclusions using `save_analysis` tool
- No JSON in prompt responses

**Example of correct flow:**
```
Agent: "I need to check BTC price first"
Agent: [calls get_price("BTC/USDT")]
Tool returns: {"price": 45000, "bid": 44995, "ask": 45005}
Agent: "Price is $45k. Now I need HTF trend context"
Agent: [calls get_ohlcv("BTC/USDT", "1D", 30)]
Tool returns: [OHLCV data...]
Agent: "1D shows clear uptrend. Now checking momentum..."
Agent: [calls rsi("BTC/USDT", "1D", 14)]
Tool returns: {"rsi": 62}
Agent: "RSI 62 - healthy momentum. I'm confident: bullish setup"
Agent: [calls save_analysis({...})]
Tool confirms: {"status": "saved", "id": "uuid"}
Agent: "Analysis complete and saved."
```

---

## How to Rewrite Prompts

### Step 1: Keep the Header (ROLE & MANDATE)

Don't change:
- Role description
- Mandate bullet points
- "You serve:" section

```markdown
# ROLE & MANDATE

You are the **Senior Market Structure Analyst** — the technical analysis backbone for trading decisions.

**Your mandate:**
- Analyze market structure across multiple timeframes
- ...
```

### Step 2: Keep REASONING APPROACH

The section explaining step-by-step thinking is GOOD. Keep it:

```markdown
# REASONING APPROACH

**Think step-by-step. Use tools iteratively to build understanding.**

1. **Assess what you know** - Start with context you have
2. **Identify gaps** - What data do you need?
3. **Call tools ONE AT A TIME** - Get price, then OHLCV, then indicators
...
```

### Step 3: Keep AVAILABLE TOOLS

Don't change the tools listing section. It's auto-generated from `{{.Tools}}`.

### Step 4: Keep QUALITY PRINCIPLES

Keep domain-specific guidelines:
- For market_analyst: HTF/LTF, confluence, risk-first
- For smc_analyst: Order blocks, FVG, liquidity
- For risk_manager: Hard limits, stress testing
- etc.

### Step 5: Keep CONVERSATION GUIDELINES

The "As you work through analysis" section is PERFECT for CoT. Keep it:

```markdown
**As you work through analysis:**

1. **State your current understanding**
   - "I see price at $42,000. Need to check HTF trend..."

2. **Explain tool choices**
   - "Calling get_ohlcv for 1D timeframe to identify HTF structure"

3. **Reflect on results**
   - "1D shows higher-highs and higher-lows since October — uptrend intact"
...
```

### Step 6: REPLACE "FINAL OUTPUT" Section

**❌ DELETE everything that looks like:**
```markdown
# FINAL OUTPUT

When you have sufficient analysis and are confident in your view, provide structured output:

```json
{
  "analysis_summary": "...",
  ...
}
```
```

**✅ REPLACE with:**

```markdown
# HOW TO COMPLETE YOUR ANALYSIS

**DO NOT return JSON. Instead, save your findings using the `save_analysis` tool.**

When you've gathered sufficient data and are confident in your conclusions:

1. **Summarize your findings verbally**
   - State your directional bias (bullish/bearish/neutral)
   - Explain key levels and confluence
   - Describe risk/reward setup

2. **Save analysis via tool**
   Call: save_analysis({
     "agent_type": "{{.AgentName}}",
     "symbol": "{{.Symbol}}",
     "analysis": {
       // Structure this based on what you actually discovered
       // Include your key findings
     }
   })

3. **Confirm completion**
   - "Analysis saved. Ready for next agent."

**Important:**
- Structure your analysis object however makes sense for your findings
- Include what you actually discovered (don't force all fields if irrelevant)
- The save_analysis tool will persist your work for other agents
```

### Step 7: Update CONSTRAINTS

In the CONSTRAINTS section, add:

```markdown
❌ **NEVER:**
- Return JSON directly in your response
- Skip saving your analysis via save_analysis tool
- Claim you're "done" without saving
```

```markdown
✅ **ALWAYS:**
- Save conclusions using save_analysis tool
- Confirm when analysis is saved and ready
```

---

## Tool Reference for Each Agent

### Available Tools for Saving Results

**All agents have access to:**
- `save_analysis` - Save completed analysis with structured findings
- `save_insight` - Save learnings/patterns for long-term memory
- `search_memory` - Search past analyses for similar scenarios
- `record_reasoning` - Explicitly log a reasoning step (optional, mainly for CoTWrapper)

### Agent-Specific Instructions

#### Analysts (market, SMC, sentiment, macro, derivatives, orderflow, onchain, correlation)

**Tool to use:** `save_analysis`

**Example:**
```javascript
save_analysis({
  "agent_type": "market_analyst", // or smc_analyst, sentiment_analyst, etc.
  "symbol": "BTC/USDT",
  "timeframe": "multiple",
  "analysis": {
    // Your findings - structure freely
    "direction": "bullish",
    "confidence": 0.80,
    "key_insight": "HTF uptrend with LTF pullback to support",
    // ... whatever you discovered
  }
})
```

#### Strategy Planner

**Tool to use:** `save_analysis`

**Example:**
```javascript
save_analysis({
  "agent_type": "strategy_planner",
  "symbol": "BTC/USDT",
  "analysis": {
    "recommendation": "enter_long",
    "consensus": ["market_analyst", "smc_analyst", "derivatives_analyst"],
    "plan": {
      "entry": [39500, 39800],
      "stop": 39000,
      "targets": [41000, 42500],
      "size_pct": 6
    },
    "rationale": "Strong technical confluence with manageable macro risk"
  }
})
```

#### Risk Manager

**Tool to use:** `save_analysis`

**Example:**
```javascript
save_analysis({
  "agent_type": "risk_manager",
  "symbol": "BTC/USDT",
  "analysis": {
    "decision": "approved" | "rejected" | "approved_with_conditions",
    "checks": {
      "position_size": "✓ within 10% limit",
      "leverage": "✓ below max",
      "correlation": "✓ acceptable"
    },
    "conditions": ["reduce size to 5% due to macro risk"],
    "reasoning": "Plan passes all checks but macro headwind warrants sizing down"
  }
})
```

#### Executor

**Tool to use:** `save_analysis`

**Example:**
```javascript
save_analysis({
  "agent_type": "executor",
  "symbol": "BTC/USDT",
  "analysis": {
    "execution_plan": "TWAP over 60 minutes",
    "venue": "binance",
    "orders": [
      {"type": "limit_post_only", "size": "0.167 BTC", "slices": 60}
    ],
    "slippage_limit": 0.05,
    "reasoning": "Good liquidity, low urgency → minimize cost via TWAP"
  }
})
```

---

## Template Variables

Prompts use Go templates. Available variables:

- `{{.AgentName}}` - Agent type (e.g., "market_analyst")
- `{{.AgentType}}` - Same as AgentName
- `{{.Symbol}}` - Trading pair (e.g., "BTC/USDT")
- `{{.Timeframe}}` - Requested timeframe if applicable
- `{{.Tools}}` - Array of available tools (auto-populated)
- `{{.MaxToolCalls}}` - Tool call limit for this agent
- `{{.AgentState}}` - Optional agent state (past performance, etc.)

**Usage:**
```markdown
Call: save_analysis({
  "agent_type": "{{.AgentName}}",
  "symbol": "{{.Symbol}}",
  ...
})
```

---

## Complete Example: market_analyst.tmpl

See [`market_analyst.tmpl`](./market_analyst.tmpl) for a fully rewritten example.

**Key changes made:**
1. ❌ Removed JSON output block (lines 124-209)
2. ✅ Added "HOW TO COMPLETE YOUR ANALYSIS" section
3. ✅ Instructions to use `save_analysis` tool
4. ✅ Updated constraints to forbid direct JSON returns

---

## Checklist for Each Prompt

When rewriting a prompt, verify:

- [ ] Removed all JSON output blocks from prompt
- [ ] Added "HOW TO COMPLETE YOUR ANALYSIS" section
- [ ] Included example `save_analysis` call
- [ ] Told agent to confirm when saved ("Analysis saved. Ready.")
- [ ] Updated CONSTRAINTS to forbid JSON returns
- [ ] Kept all domain expertise (HTF/LTF, SMC concepts, risk rules, etc.)
- [ ] Kept CONVERSATION GUIDELINES (think aloud, reflect on results, etc.)
- [ ] Template variables use `{{.Variable}}` syntax

---

## Testing Your Rewritten Prompt

After rewriting:

1. **Check template syntax:**
   ```bash
   make test  # Will catch template errors
   ```

2. **Verify tool access:**
   - Check `internal/agents/tool_assignments.go`
   - Ensure agent's category includes "memory" for save_analysis access

3. **Expected behavior:**
   ```
   User: "Analyze BTC/USDT"
   Agent: "I'll start by checking current price..."
   Agent: [calls get_price]
   Agent: "Price is $45k. Now checking HTF trend..."
   Agent: [calls get_ohlcv("1D")]
   Agent: "Clear uptrend. Checking momentum..."
   Agent: [calls rsi]
   Agent: "Bullish setup confirmed. Saving analysis..."
   Agent: [calls save_analysis({...})]
   Agent: "Analysis saved. Ready for strategy_planner."
   ```

---

## Common Mistakes to Avoid

### ❌ Mistake 1: Leaving JSON in prompt
```markdown
# FINAL OUTPUT
```json
{ "analysis": ... }
```
```
**Fix:** Remove entirely, replace with save_analysis instructions

### ❌ Mistake 2: Not telling agent HOW to finish
Agent doesn't know to call save_analysis.

**Fix:** Explicit instructions in "HOW TO COMPLETE YOUR ANALYSIS"

### ❌ Mistake 3: Over-specifying analysis structure
```markdown
Call save_analysis with EXACTLY these fields: direction, confidence, htf_trend, ltf_flow...
```

**Fix:** Give flexibility. Agent should structure based on what it found.

### ❌ Mistake 4: Forgetting template variables
```markdown
"agent_type": "market_analyst"  // Hardcoded
```

**Fix:**
```markdown
"agent_type": "{{.AgentName}}"  // Dynamic
```

---

## FAQ

**Q: Should I change domain-specific sections (like HTF/LTF for market_analyst)?**
A: NO. Keep all domain expertise intact. Only change output format.

**Q: What if agent has unique output needs?**
A: Adapt the `save_analysis` call structure. Example object can be customized per agent.

**Q: Do all agents use save_analysis?**
A: Yes. Even risk_manager (saves approval/rejection decision) and executor (saves execution plan).

**Q: What about position_manager monitoring live positions?**
A: Still uses save_analysis to record monitoring updates and decisions.

**Q: Can agents save multiple times during analysis?**
A: Yes! They can use `save_insight` for learnings discovered along the way, then `save_analysis` for final conclusion.

---

## Next Steps

1. **Review the example:** [`market_analyst.tmpl`](./market_analyst.tmpl)
2. **Pick a prompt to rewrite** (start with simpler ones like smc_analyst)
3. **Follow the pattern:**
   - Keep header & domain expertise
   - Remove JSON output
   - Add save_analysis instructions
4. **Test locally** before committing

---

**Questions?** Check `docs/DEVELOPMENT_PLAN.md` Phase 1 or ask in the PR.

