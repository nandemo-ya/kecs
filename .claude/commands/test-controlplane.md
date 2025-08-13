Run the controlplane unit tests using Ginkgo (if available) or go test as a fallback.

Execute:
```bash
command -v ginkgo >/dev/null 2>&1 && ginkgo -r -v || go test -v -race ./...