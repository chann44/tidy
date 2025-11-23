#!/bin/bash

# Benchmark script to compare Tidy vs Bun vs npm vs pnpm

echo "🚀 Package Manager Performance Benchmark"
echo "=========================================="
echo ""

# Test packages - a realistic Next.js project
TEST_PACKAGES="react react-dom next typescript @types/react @types/node tailwindcss postcss autoprefixer"

# Create a temporary directory for testing
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

echo "📁 Test directory: $TEMP_DIR"
echo ""

# Initialize package.json
cat > package.json <<EOF
{
  "name": "benchmark-test",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "next": "^14.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/react": "^18.0.0",
    "@types/node": "^20.0.0",
    "tailwindcss": "^3.0.0",
    "postcss": "^8.0.0",
    "autoprefixer": "^10.0.0"
  }
}
EOF

echo "📦 Test packages: $TEST_PACKAGES"
echo ""

# Function to benchmark a package manager
benchmark() {
    local pm=$1
    local name=$2
    
    echo "⏱️  Benchmarking $name..."
    
    # Clean install
    rm -rf node_modules package-lock.json yarn.lock pnpm-lock.yaml bun.lockb
    
    # Cold install (no cache)
    if [ "$pm" = "tidy" ]; then
        rm -rf ~/.tidy-cache
    elif [ "$pm" = "bun" ]; then
        rm -rf ~/.bun/install/cache
    elif [ "$pm" = "pnpm" ]; then
        pnpm store prune
    elif [ "$pm" = "npm" ]; then
        npm cache clean --force
    fi
    
    local start=$(date +%s%N)
    
    if [ "$pm" = "tidy" ]; then
        tidy install > /dev/null 2>&1
    elif [ "$pm" = "bun" ]; then
        bun install > /dev/null 2>&1
    elif [ "$pm" = "pnpm" ]; then
        pnpm install > /dev/null 2>&1
    elif [ "$pm" = "npm" ]; then
        npm install > /dev/null 2>&1
    fi
    
    local end=$(date +%s%N)
    local cold_time=$(( (end - start) / 1000000 ))
    
    # Clean for warm install
    rm -rf node_modules
    
    # Warm install (with cache)
    start=$(date +%s%N)
    
    if [ "$pm" = "tidy" ]; then
        tidy install > /dev/null 2>&1
    elif [ "$pm" = "bun" ]; then
        bun install > /dev/null 2>&1
    elif [ "$pm" = "pnpm" ]; then
        pnpm install > /dev/null 2>&1
    elif [ "$pm" = "npm" ]; then
        npm install > /dev/null 2>&1
    fi
    
    local end=$(date +%s%N)
    local warm_time=$(( (end - start) / 1000000 ))
    
    echo "  ✓ Cold install: ${cold_time}ms"
    echo "  ✓ Warm install: ${warm_time}ms"
    echo ""
    
    # Return results
    echo "$name,$cold_time,$warm_time"
}

# Run benchmarks
echo "🏁 Starting benchmarks..."
echo ""

results=""

# Check which package managers are available
if command -v tidy &> /dev/null; then
    results+=$(benchmark "tidy" "Tidy")
    results+=$'\n'
fi

if command -v bun &> /dev/null; then
    results+=$(benchmark "bun" "Bun")
    results+=$'\n'
fi

if command -v pnpm &> /dev/null; then
    results+=$(benchmark "pnpm" "pnpm")
    results+=$'\n'
fi

if command -v npm &> /dev/null; then
    results+=$(benchmark "npm" "npm")
    results+=$'\n'
fi

# Display results table
echo "📊 Results Summary"
echo "=================="
echo ""
printf "%-15s %-20s %-20s\n" "Package Manager" "Cold Install (ms)" "Warm Install (ms)"
printf "%-15s %-20s %-20s\n" "---------------" "-------------------" "-------------------"

echo "$results" | while IFS=',' read -r name cold warm; do
    if [ -n "$name" ]; then
        printf "%-15s %-20s %-20s\n" "$name" "$cold" "$warm"
    fi
done

echo ""
echo "✅ Benchmark complete!"
echo ""

# Cleanup
cd -
rm -rf "$TEMP_DIR"
