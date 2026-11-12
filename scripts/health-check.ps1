# ============================================================================
# MAICIVY - Infrastructure Health Check Script (PowerShell - Windows)
# ============================================================================
# Usage: .\scripts\health-check.ps1
# ============================================================================

$ErrorActionPreference = "Stop"

# Configuration
$DOCKER_COMPOSE_FILE = "docker-compose.yml"
$REQUIRED_SERVICES = @("postgres", "redis", "backend", "frontend")
$MAX_RETRIES = 10
$RETRY_DELAY = 2

function Write-StatusOk {
    param([string]$message)
    Write-Host "✓ $message" -ForegroundColor Green
}

function Write-StatusError {
    param([string]$message)
    Write-Host "✗ $message" -ForegroundColor Red
}

function Write-StatusWarn {
    param([string]$message)
    Write-Host "⟳ $message" -ForegroundColor Yellow
}

function Write-Header {
    param([string]$message)
    Write-Host "========================================" -ForegroundColor Blue
    Write-Host $message -ForegroundColor Blue
    Write-Host "========================================" -ForegroundColor Blue
}

# Check Docker
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-StatusError "Docker is not installed"
    exit 1
}
Write-StatusOk "Docker is installed"

# Check Docker Compose
if (-not (Get-Command docker-compose -ErrorAction SilentlyContinue)) {
    Write-StatusError "Docker Compose is not installed"
    exit 1
}
Write-StatusOk "Docker Compose is installed"

# Main checks
Write-Header "MAICIVY Infrastructure Health Check"

# Check docker-compose.yml exists
if (-not (Test-Path $DOCKER_COMPOSE_FILE)) {
    Write-StatusError "$DOCKER_COMPOSE_FILE not found"
    exit 1
}
Write-StatusOk "$DOCKER_COMPOSE_FILE found"

# Check service health
Write-Host "`nChecking service health..." -ForegroundColor Blue

$all_healthy = $true
foreach ($service in $REQUIRED_SERVICES) {
    $retry_count = 0
    $is_healthy = $false

    while ($retry_count -lt $MAX_RETRIES) {
        try {
            $health = docker inspect -f '{{.State.Health.Status}}' maicivy-$service 2>$null
            $state = docker inspect -f '{{.State.Running}}' maicivy-$service 2>$null

            if ($health -eq "healthy" -or $state -eq "true") {
                Write-StatusOk "$service"
                $is_healthy = $true
                break
            }
        }
        catch {
            # Ignore errors
        }

        Write-StatusWarn "$service not ready ($($retry_count + 1)/$MAX_RETRIES)"
        $retry_count++
        Start-Sleep -Seconds $RETRY_DELAY
    }

    if (-not $is_healthy) {
        Write-StatusError "$service"
        $all_healthy = $false
    }
}

# Show container status
Write-Host "`nContainer status:" -ForegroundColor Blue
docker-compose ps

if ($all_healthy) {
    Write-Host "`nAll services are healthy!" -ForegroundColor Green
    Write-Host "`nAccess points:" -ForegroundColor Blue
    Write-Host "  Frontend:  http://localhost:3000"
    Write-Host "  Backend:   http://localhost:8080"
    Write-Host "  PostgreSQL: localhost:5432"
    Write-Host "  Redis:     localhost:6379"
    exit 0
} else {
    Write-Host "`nSome services are not healthy" -ForegroundColor Red
    exit 1
}
