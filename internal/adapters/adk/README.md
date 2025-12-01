<!-- @format -->

# ADK Adapters

This package contains adapters that bridge our internal implementations with Google ADK interfaces.

## Model Adapter

**Purpose:** Connects our unified AI provider interface (`ai.ChatProvider`) to ADK's `model.LLM` interface.

**Why needed:**

- We have multiple AI providers (Claude, OpenAI, DeepSeek) behind a single `ChatProvider` interface
- ADK expects `model.LLM` with `GenerateContent()` method
- Adapter translates between our API and ADK's API

**Flow:**

```
ADK Agent → model.LLM.GenerateContent() → ModelAdapter → ai.ChatProvider.Chat() → Claude/OpenAI/DeepSeek
```

## No Tool Adapter

We don't need a tool adapter because:

- ADK provides `tool/functiontool` package
- We use `functiontool.New()` directly to wrap our tool functions
- No translation needed - ADK handles it natively

**Flow:**

```
ADK Agent → tool.Tool → functiontool wrapper → our tool.Execute()
```



