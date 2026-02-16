# Model Fallback Chains

OpenClaw supports fallback chains for LLM providers via the `llmConfig` section of `openclaw.json`. When the primary provider is unavailable or returns an error, the application automatically tries the next provider in the chain.

## How It Works

Fallback logic is **application behavior** — the operator does not intercept or manage LLM calls. The operator's role is to deliver the `openclaw.json` config and inject the required API keys via `envFrom` or `env`.

The `llmConfig` section in `openclaw.json` accepts an ordered list of provider configurations. When a request fails (timeout, rate limit, 5xx), OpenClaw retries with the next provider in the list.

## Example CR

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-assistant
spec:
  config:
    raw:
      llmConfig:
        - provider: anthropic
          model: claude-sonnet-4-5-20250929
        - provider: openai
          model: gpt-4o
          baseURL: https://api.openai.com/v1
        - provider: google
          model: gemini-2.0-flash

  envFrom:
    - secretRef:
        name: ai-provider-keys
```

The Secret `ai-provider-keys` should contain all required API keys:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ai-provider-keys
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-..."
  OPENAI_API_KEY: "sk-..."
  GOOGLE_AI_API_KEY: "AIza..."
```

## Required API Keys by Provider

| Provider   | Environment Variable           |
|------------|--------------------------------|
| Anthropic  | `ANTHROPIC_API_KEY`            |
| OpenAI     | `OPENAI_API_KEY`               |
| Google AI  | `GOOGLE_AI_API_KEY`            |
| Azure OpenAI | `AZURE_OPENAI_API_KEY` + `AZURE_OPENAI_ENDPOINT` |
| AWS Bedrock | `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` |
| Mistral    | `MISTRAL_API_KEY`              |
| Groq       | `GROQ_API_KEY`                 |
| DeepSeek   | `DEEPSEEK_API_KEY`             |
| OpenRouter | `OPENROUTER_API_KEY`           |

## Notes

- The operator webhook warns if no known provider API keys are detected in `envFrom` or `env`.
- Fallback ordering matters — place your preferred (fastest/cheapest) provider first.
- Each provider in the chain must have its API key available in the pod environment.
- Rate limits and quotas are per-provider. A fallback chain spreads load across providers during outages but does not pool quotas.
