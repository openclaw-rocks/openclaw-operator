# Custom AI Providers

This guide covers common patterns for connecting OpenClaw to self-hosted or alternative AI providers.

## Ollama as a Sidecar

Run Ollama alongside OpenClaw in the same pod. This is the simplest option when you want local model inference without network hops.

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: local-llm
spec:
  config:
    raw:
      llmConfig:
        - provider: openai
          model: llama3.2
          baseURL: http://localhost:11434/v1

  sidecars:
    - name: ollama
      image: ollama/ollama:latest
      ports:
        - containerPort: 11434
          protocol: TCP
      volumeMounts:
        - name: ollama-models
          mountPath: /root/.ollama

  sidecarVolumes:
    - name: ollama-models
      emptyDir:
        sizeLimit: 20Gi

  # No external API keys needed for local inference
  env:
    - name: OPENAI_API_KEY
      value: "not-needed"

  resources:
    requests:
      cpu: "2"
      memory: 8Gi
    limits:
      cpu: "8"
      memory: 16Gi

  security:
    networkPolicy:
      enabled: true
      # No egress needed for local-only inference
```

### GPU Support

For GPU-accelerated Ollama, add resource limits and node selectors:

```yaml
spec:
  sidecars:
    - name: ollama
      image: ollama/ollama:latest
      resources:
        limits:
          nvidia.com/gpu: "1"
      volumeMounts:
        - name: ollama-models
          mountPath: /root/.ollama

  availability:
    nodeSelector:
      gpu: "true"
    tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
```

## Ollama as an External Service

When Ollama runs as a separate Deployment or on bare metal, point OpenClaw to it via config and allow egress to the service.

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: external-ollama
spec:
  config:
    raw:
      llmConfig:
        - provider: openai
          model: llama3.2
          baseURL: http://ollama.inference.svc:11434/v1

  env:
    - name: OPENAI_API_KEY
      value: "not-needed"

  security:
    networkPolicy:
      enabled: true
      additionalEgress:
        - to:
            - namespaceSelector:
                matchLabels:
                  kubernetes.io/metadata.name: inference
              podSelector:
                matchLabels:
                  app: ollama
          ports:
            - protocol: TCP
              port: 11434
```

## vLLM via OpenAI-Compatible API

[vLLM](https://docs.vllm.ai/) exposes an OpenAI-compatible API. Configure it the same way as Ollama:

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: vllm-instance
spec:
  config:
    raw:
      llmConfig:
        - provider: openai
          model: meta-llama/Llama-3.2-8B-Instruct
          baseURL: http://vllm.inference.svc:8000/v1

  env:
    - name: OPENAI_API_KEY
      value: "not-needed"

  security:
    networkPolicy:
      enabled: true
      additionalEgress:
        - to:
            - namespaceSelector:
                matchLabels:
                  kubernetes.io/metadata.name: inference
              podSelector:
                matchLabels:
                  app: vllm
          ports:
            - protocol: TCP
              port: 8000
```

## NetworkPolicy Considerations

The default NetworkPolicy allows egress only on port 443 (HTTPS) and port 53 (DNS). When using custom providers on non-standard ports, you must add egress rules:

| Provider Setup        | Port  | Solution                              |
|-----------------------|-------|---------------------------------------|
| Ollama sidecar        | 11434 | No egress needed (localhost)          |
| Ollama external       | 11434 | `additionalEgress` with pod selector  |
| vLLM external         | 8000  | `additionalEgress` with pod selector  |
| Custom HTTPS endpoint | 443   | Already allowed by default            |
| Custom non-443 HTTPS  | 8443  | `additionalEgress` with CIDR or selector |

## Hybrid Fallback (Local + Cloud)

Combine a local provider with cloud fallbacks for resilience:

```yaml
spec:
  config:
    raw:
      llmConfig:
        - provider: openai
          model: llama3.2
          baseURL: http://localhost:11434/v1
        - provider: anthropic
          model: claude-sonnet-4-5-20250929

  envFrom:
    - secretRef:
        name: cloud-api-keys

  env:
    - name: OPENAI_API_KEY
      value: "not-needed"
```

This tries local Ollama first and falls back to Anthropic Claude if the local model is unavailable or errors. See [Model Fallback Chains](model-fallback.md) for details on fallback behavior.
