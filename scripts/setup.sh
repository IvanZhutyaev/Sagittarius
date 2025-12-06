#!/bin/bash

set -e

echo "🚀 Setting up Sagittarius project..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Generate protobuf code
echo "📦 Generating protobuf code..."
chmod +x scripts/generate-proto.sh
./scripts/generate-proto.sh || echo "⚠️  Protobuf generation failed, continuing..."

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
go mod download || echo "⚠️  Go mod download failed, continuing..."

# Start infrastructure
echo "🐳 Starting infrastructure services..."
docker-compose up -d postgres redis clickhouse zookeeper kafka

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Create Kafka topics
echo "📨 Creating Kafka topics..."
chmod +x scripts/create-kafka-topics.sh
./scripts/create-kafka-topics.sh || echo "⚠️  Kafka topics creation failed, continuing..."

# Start all services
echo "🚀 Starting all services..."
docker-compose up -d

echo "✅ Setup complete!"
echo ""
echo "📊 Services:"
echo "  - API Gateway: http://localhost:8080"
echo "  - Prometheus: http://localhost:9090"
echo "  - Grafana: http://localhost:3000 (admin/admin)"
echo "  - Jaeger: http://localhost:16686"
echo ""
echo "Check status with: docker-compose ps"
echo "View logs with: docker-compose logs -f"

