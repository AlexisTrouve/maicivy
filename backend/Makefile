.PHONY: help build run test test-short test-coverage clean fmt vet lint dev docker-build

# Variables
BINARY_NAME=main
GO=go
GOFLAGS=-v

help: ## Afficher cette aide
	@echo "Commandes disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build le binaire
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/main.go

run: ## Lancer l'application
	$(GO) run ./cmd/main.go

test: ## Lancer tous les tests
	$(GO) test $(GOFLAGS) ./...

test-short: ## Lancer les tests unitaires uniquement (rapide)
	$(GO) test $(GOFLAGS) -short ./...

test-coverage: ## Générer le coverage
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

clean: ## Nettoyer les fichiers générés
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	$(GO) clean

fmt: ## Formater le code
	$(GO) fmt ./...

vet: ## Analyser le code
	$(GO) vet ./...

lint: fmt vet ## Linter complet

dev: ## Mode développement (avec hot reload - nécessite air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

docker-build: ## Build l'image Docker
	docker build -t maicivy-backend:latest .

deps: ## Installer/mettre à jour les dépendances
	$(GO) mod download
	$(GO) mod tidy

mod-vendor: ## Créer le dossier vendor
	$(GO) mod vendor
