#!/bin/bash

# generate-api-clients.sh
# Auto-generate API clients from OpenAPI specification

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OPENAPI_SPEC="$PROJECT_ROOT/docs/api/openapi.yaml"
OUTPUT_DIR="$PROJECT_ROOT/generated-clients"

echo "🚀 Generating API clients from OpenAPI spec..."
echo "   Spec: $OPENAPI_SPEC"
echo "   Output: $OUTPUT_DIR"
echo ""

# Check if OpenAPI spec exists
if [ ! -f "$OPENAPI_SPEC" ]; then
    echo "❌ Error: OpenAPI spec not found at $OPENAPI_SPEC"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# ============================================
# 1. Generate TypeScript/JavaScript Client
# ============================================
echo "📦 Generating TypeScript client..."

if command -v npx &> /dev/null; then
    npx openapi-typescript "$OPENAPI_SPEC" \
        --output "$OUTPUT_DIR/typescript/api.ts"

    echo "   ✅ TypeScript types generated"
else
    echo "   ⚠️  npx not found, skipping TypeScript generation"
    echo "   Install: npm install -g openapi-typescript"
fi

# ============================================
# 2. Generate Go Client
# ============================================
echo ""
echo "🐹 Generating Go client..."

if command -v oapi-codegen &> /dev/null; then
    oapi-codegen -generate types \
        -package apiclient \
        "$OPENAPI_SPEC" > "$OUTPUT_DIR/go/types.go"

    oapi-codegen -generate client \
        -package apiclient \
        "$OPENAPI_SPEC" > "$OUTPUT_DIR/go/client.go"

    echo "   ✅ Go client generated"
else
    echo "   ⚠️  oapi-codegen not found, skipping Go generation"
    echo "   Install: go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest"
fi

# ============================================
# 3. Generate Python Client
# ============================================
echo ""
echo "🐍 Generating Python client..."

if command -v openapi-generator-cli &> /dev/null; then
    openapi-generator-cli generate \
        -i "$OPENAPI_SPEC" \
        -g python \
        -o "$OUTPUT_DIR/python" \
        --package-name maicivy_api

    echo "   ✅ Python client generated"
else
    echo "   ⚠️  openapi-generator-cli not found, skipping Python generation"
    echo "   Install: npm install -g @openapitools/openapi-generator-cli"
fi

# ============================================
# 4. Generate Rust Client (Optional)
# ============================================
echo ""
echo "🦀 Generating Rust client (optional)..."

if command -v openapi-generator-cli &> /dev/null; then
    openapi-generator-cli generate \
        -i "$OPENAPI_SPEC" \
        -g rust \
        -o "$OUTPUT_DIR/rust" \
        --package-name maicivy-api

    echo "   ✅ Rust client generated"
else
    echo "   ⚠️  Skipping Rust generation"
fi

# ============================================
# Summary
# ============================================
echo ""
echo "======================================"
echo "✅ API Client Generation Complete"
echo "======================================"
echo ""
echo "Generated clients in:"
echo "   - TypeScript: $OUTPUT_DIR/typescript/"
echo "   - Go:         $OUTPUT_DIR/go/"
echo "   - Python:     $OUTPUT_DIR/python/"
echo "   - Rust:       $OUTPUT_DIR/rust/ (optional)"
echo ""
echo "📖 Usage examples:"
echo ""
echo "TypeScript:"
echo "   import { paths } from './generated-clients/typescript/api';"
echo "   type CVResponse = paths['/api/v1/cv']['get']['responses']['200']['content']['application/json'];"
echo ""
echo "Go:"
echo "   import \"maicivy/generated-clients/go/apiclient\""
echo "   client, _ := apiclient.NewClient(\"http://localhost:5000\")"
echo ""
echo "Python:"
echo "   from maicivy_api import ApiClient, Configuration"
echo "   config = Configuration(host='http://localhost:5000')"
echo "   client = ApiClient(config)"
echo ""
echo "======================================"

# Create README in output directory
cat > "$OUTPUT_DIR/README.md" << 'EOF'
# Generated API Clients

Auto-generated API clients from OpenAPI specification.

## Installation & Usage

### TypeScript

```typescript
import { paths } from './typescript/api';

type CVResponse = paths['/api/v1/cv']['get']['responses']['200']['content']['application/json'];

// Use with fetch
const response = await fetch('http://localhost:5000/api/v1/cv?theme=backend');
const cv: CVResponse = await response.json();
```

### Go

```go
import (
    apiclient "maicivy/generated-clients/go"
)

func main() {
    client, err := apiclient.NewClient("http://localhost:5000")
    if err != nil {
        panic(err)
    }

    cv, err := client.GetCV(context.Background(), &apiclient.GetCVParams{
        Theme: "backend",
    })
}
```

### Python

```python
from maicivy_api import ApiClient, Configuration
from maicivy_api.api import cv_api

config = Configuration(host='http://localhost:5000')
client = ApiClient(config)
api = cv_api.CVApi(client)

cv = api.get_cv(theme='backend')
print(cv)
```

## Regenerating Clients

Run the generation script whenever the OpenAPI spec changes:

```bash
bash scripts/generate-api-clients.sh
```

## Dependencies

- **TypeScript**: `openapi-typescript`
- **Go**: `oapi-codegen`
- **Python**: `openapi-generator-cli`

## Documentation

See [docs/api/API_OVERVIEW.md](../../docs/api/API_OVERVIEW.md) for complete API documentation.
EOF

echo ""
echo "📄 README created at $OUTPUT_DIR/README.md"
echo ""
echo "🎉 Done!"
