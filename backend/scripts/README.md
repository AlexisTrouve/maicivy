# Backend Scripts

Ces scripts utilitaires ont des build tags pour éviter les conflits lors de la compilation. Pour les exécuter, utilisez la commande suivante :

## Utilisation

### Initialiser la base de données
```bash
go run -tags scripts scripts/init_db.go
```

### Exécuter les migrations
```bash
go run -tags scripts scripts/migrate.go [up|down|version|force <version>]
```

### Peupler la base de données avec des données de test
```bash
go run -tags scripts scripts/seed.go
```

## Pourquoi des build tags ?

Les build tags `//go:build scripts` empêchent ces fichiers d'être compilés avec le reste du code lors d'un `go build ./...`, évitant ainsi les conflits de multiples fonctions `main()`.

## Impact sur les scans de sécurité

Ces scripts sont **exclus des scans de sécurité** dans GitHub Actions car ils ne font pas partie du code de production :

- **gosec** : `--exclude-dir=scripts`
- **govulncheck** : Utilise `go list ./... | grep -v '/scripts$'`

Cette exclusion est intentionnelle et documentée dans `.github/workflows/security-scan.yml`.

## Scripts disponibles

| Script | Description | Usage |
|--------|-------------|-------|
| `init_db.go` | Initialise la base de données (création + migrations) | Setup initial |
| `migrate.go` | Gestion des migrations manuelles | Développement |
| `seed.go` | Peuple la DB avec des données de test | Développement |

**Note:** Ces scripts sont des outils de développement et ne sont jamais déployés en production.
