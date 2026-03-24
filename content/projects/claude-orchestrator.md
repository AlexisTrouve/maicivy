---
title: "ClaudeOrchestrator"
category: ai
technologies: [Python, asyncio, Pydantic, Git Worktrees, Claude Code, Gitea API, GitHub API]
featured: true
in_progress: true
github_language: Python
catchphrase: "N workers Claude Code en parallèle → review IA → PR automatique, ~0,25€ par tâche"
---

## Description

Transforme une file de tâches en pull requests reviewées et testées sans intervention manuelle. ClaudeOrchestrator dispatche N workers Claude Code en parallèle, chacun isolé dans son propre git worktree, puis enchaîne builds automatisés, code review IA et suites de tests avant de créer des PRs via l'API Gitea ou GitHub.

## Description Technique

Système de retry avec boucle de feedback : les commentaires du reviewer et les logs d'échec de tests sont injectés dans le prompt du prochain worker, l'IA apprend de ses propres erreurs. Circuit breaker qui stoppe le pipeline après 3 échecs consécutifs. Trois sources de tâches connectables : fichiers YAML, GitHub Issues, API ProjectMind. Isolation via git worktrees pour zéro conflit entre workers parallèles. Architecture Pydantic v2, asyncio Python 3.12.

## Description Fonctionnelle

Outil d'automatisation de développement IA : soumettre des tâches, obtenir des PRs reviewées et testées. ~0,25€ par tâche, ~70 secondes de traitement. Chaque décision loggée, chaque diff sauvegardé, chaque review traçable.

## Stats

- **2 900+** lignes Python
- **67+** tests
- **~0,25€** par tâche
- **6** étapes de pipeline
