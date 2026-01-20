# Synchronisation GitHub - Guide

Ce document décrit le processus de synchronisation entre le repository privé Gitea et le repository public GitHub.

## Architecture Dual-Repository

```
┌─────────────────────────────┐         ┌─────────────────────────────┐
│  Gitea (origin/main)        │         │  GitHub (github/main)       │
│  git.etheryale.com          │         │  github.com                 │
├─────────────────────────────┤         ├─────────────────────────────┤
│ ✅ Historique complet       │         │ ✅ Historique propre        │
│ ✅ Mentions Claude          │   ──►   │ ❌ Sans mentions Claude     │
│ ✅ Audits internes          │         │ ❌ Sans audits internes     │
│ ✅ Workflow docs            │         │ ✅ Code public              │
│ 🔒 Privé                    │         │ 🌐 Public                   │
└─────────────────────────────┘         └─────────────────────────────┘
```

## Processus de synchronisation

### Méthode recommandée: Skill Claude

Si vous utilisez Claude Code, utilisez le skill intégré:
```bash
/sync-github
```

Le skill se trouve dans `.claude/skills/sync-github.md` (local uniquement)

### Méthode manuelle

#### 1. Vérifications préliminaires
```bash
# Vérifier qu'on est sur main
git checkout main
git status

# Voir les commits à synchroniser
git log github/main..main --oneline
```

#### 2. Créer la branche de sync
```bash
# Fetch GitHub
git fetch github

# Créer branche public depuis github/main
git checkout -B public github/main
```

#### 3. Cherry-pick et nettoyer
```bash
# Cherry-pick le commit (sans committer)
git cherry-pick --no-commit <commit-hash>

# IMPORTANT: Retirer les fichiers internes
git rm --cached .github/SECURITY_FIXES_2026-01-20.md 2>/dev/null || true

# Commit SANS mention Claude
git commit -m "<message propre sans Co-Authored-By>"
```

#### 4. Push vers GitHub
```bash
git push github public:main
```

#### 5. Nettoyage
```bash
git checkout main
git branch -D public
```

## Fichiers exclus de GitHub

Ces fichiers ne doivent **JAMAIS** être pushés sur GitHub:

### Audits internes
- `.github/SECURITY_FIXES_2026-01-20.md`

### Documentation workflow (non commités)
- `AUTO_SYNC_PLAN.txt`
- `GIT_WORKFLOW.md`
- `START_HERE.md`
- `SYNC_README.md`
- `TEST_SYNC.md`

### Scripts de sync (non commités)
- `auto-sync.sh`
- `dry-run-test.sh`
- `sync-to-github*.ps1`
- `sync-to-github*.sh`

## Règles importantes

### ✅ À FAIRE
- Retirer toutes les mentions "Co-Authored-By: Claude"
- Vérifier qu'aucun fichier interne n'est inclus
- Utiliser des messages de commit professionnels
- Tester que les GitHub Actions passent

### ❌ À NE PAS FAIRE
- Ne jamais force push sans `--force-with-lease`
- Ne jamais pusher les audits internes
- Ne jamais pusher avec des mentions Claude
- Ne jamais pusher les scripts de sync

## Troubleshooting

### Erreur: "untracked working tree files would be overwritten"
```bash
git stash --include-untracked
git checkout <branch>
git stash pop
```

### Historiques divergés
```bash
# Option sûre
git push github public:main --force-with-lease

# Si vous êtes sûr
git push github public:main --force
```

## Automatisation future

Un script d'automatisation pourrait être créé pour:
1. Détecter automatiquement les nouveaux commits
2. Nettoyer les mentions Claude
3. Exclure les fichiers internes
4. Créer un commit propre
5. Pusher vers GitHub

**Note:** Pour l'instant, le processus reste manuel pour garder le contrôle total.

## Contact

Pour toute question sur ce workflow:
- Voir `.claude/skills/sync-github.md` (guide détaillé local)
- Vérifier `docs/GITHUB_SYNC.md` (ce fichier)
