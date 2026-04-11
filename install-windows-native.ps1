# Installation PostgreSQL + Redis sur Windows (sans Docker)
# À exécuter en PowerShell ADMIN (clic droit > "Exécuter en tant qu'administrateur")

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Installation PostgreSQL + Redis" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Installer PostgreSQL 14
Write-Host "Installation PostgreSQL 14..." -ForegroundColor Yellow
choco install postgresql14 -y --params "/Password:postgres"

# Installer Redis
Write-Host "Installation Redis..." -ForegroundColor Yellow
choco install redis-64 -y

Write-Host ""
Write-Host "================================" -ForegroundColor Green
Write-Host "Installation terminée!" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green
Write-Host ""

# Afficher les versions
Write-Host "Vérification des installations:" -ForegroundColor Cyan
& 'C:\Program Files\PostgreSQL\14\bin\psql.exe' --version
& 'C:\Program Files\Redis\redis-cli.exe' --version

Write-Host ""
Write-Host "Prochaines étapes:" -ForegroundColor Yellow
Write-Host "1. Redémarrer le terminal Git Bash" -ForegroundColor White
Write-Host "2. Créer la base de données: psql -U postgres -c 'CREATE DATABASE maicivy_db'" -ForegroundColor White
Write-Host "3. Créer l'utilisateur: psql -U postgres -c ""CREATE USER maicivy WITH PASSWORD 'maicivy_password'""" -ForegroundColor White
Write-Host "4. Donner les permissions: psql -U postgres -c 'GRANT ALL PRIVILEGES ON DATABASE maicivy_db TO maicivy'" -ForegroundColor White
Write-Host "5. Démarrer Redis: redis-server" -ForegroundColor White
Write-Host "6. Lancer le backend: cd backend && go run cmd/main.go" -ForegroundColor White
