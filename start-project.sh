#!/bin/bash

echo "================================"
echo "Starting maicivy Backend"
echo "================================"

# Check if Docker is running
if ! docker ps > /dev/null 2>&1; then
    echo "⏳ Starting Docker service..."
    sudo service docker start
    sleep 2
fi

# Navigate to project directory
cd "$(dirname "$0")"

echo "⏳ Starting PostgreSQL and Redis..."
docker-compose up -d postgres redis

echo "⏳ Waiting for services to be ready (30 seconds)..."
sleep 30

# Check if containers are running
echo ""
echo "📊 Container status:"
docker-compose ps

# Check if database is initialized
echo ""
echo "🔍 Checking database..."
TABLES=$(docker exec maicivy-postgres psql -U maicivy -d maicivy_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")

if [ "$TABLES" -lt 5 ]; then
    echo "⏳ Loading seed data..."
    docker exec -i maicivy-postgres psql -U maicivy -d maicivy_db < backend/migrations/seed_data.sql
    echo "✅ Seed data loaded!"
else
    echo "✅ Database already initialized (found $TABLES tables)"
fi

echo ""
echo "================================"
echo "✅ Infrastructure ready!"
echo "================================"
echo ""
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
echo ""
echo "To start the backend:"
echo "  cd backend"
echo "  go run cmd/main.go"
echo ""
echo "API will be available at: http://localhost:8080"
echo "Swagger UI: http://localhost:8080/api/docs"
echo ""
