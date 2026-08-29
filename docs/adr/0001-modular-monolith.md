# ADR 0001: Modular monolith plus worker

Status: accepted for bootstrap.

Use one Go API module with explicit domain boundaries and one worker binary. This minimizes operational complexity while preserving interfaces and an outbox seam for later extraction. Microservices, Kubernetes, event sourcing, and GraphQL are rejected for the initial product.
