

## Architecture Overview:
```
┌─────────────────────────────────────┐
│  Agent Process (main)               │
│  - Reads configs                    │
│  - Starts server processes          │
│  - Talks to LLM                     │
│  - Routes tool calls via HTTP       │
└─────────────────────────────────────┘
         │ HTTP                │ HTTP
         ↓                     ↓
┌──────────────────┐  ┌──────────────────┐
│ Python Server    │  │ Node Server      │
│ Port: 8080       │  │ Port: 8081       │
│ /execute/tool1   │  │ /execute/toolX   │
│ /execute/tool2   │  │ /execute/toolY   │
└──────────────────┘  └──────────────────┘