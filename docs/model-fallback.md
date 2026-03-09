# Model Fallback Chains

OpenClaw supports configuring multiple AI providers. Provider selection and fallback behavior are controlled entirely through environment variables - the operator's role is to inject the required API keys via `envFrom` or `env`.

## How It Works

Fallback logic is **application behavior** - the operator does not intercept or manage LLM calls. The operator delivers the `openclaw.json` config and injects the required API keys.

When multiple provider API keys are present in the environment, OpenClaw can fall back between providers when a request fails (timeout, rate limit, 5xx).

## Example CR

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-assistant
spec:
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
- Each provider must have its API key available in the pod environment.
- Rate limits and quotas are per-provider. A fallback chain spreads load across providers during outages but does not pool quotas.
