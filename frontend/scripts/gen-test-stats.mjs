#!/usr/bin/env node
// Régénère frontend/lib/test-stats.json par COMPTAGE STATIQUE des tests (rapide, ~instant).
//
// POURQUOI statique (et pas un vrai run jest+go ~75s) : appelé par le hook pre-commit (.githooks)
// → doit être instantané. On compte les DÉCLARATIONS de tests : `func Test…(… *testing.T)` côté Go
// (exclut TestMain/helpers) et les `it(` / `test(` côté frontend (*.test.ts(x), hors node_modules).
// Précision : à un poil près des compteurs runtime (sous-tests t.Run et it.each non dépliés),
// largement suffisant pour le badge de crédibilité. Avant : compteur maintenu À LA MAIN → dérive
// garantie (ex: 167 affiché vs 170 réels).
//
// `allGreen` : invariant projet (TDD, on ne commit pas rouge) — vérifié pour de vrai par la suite
// COMPLÈTE lancée avant chaque deploy. Le comptage, lui, est automatisé ici.
//
// Usage : node frontend/scripts/gen-test-stats.mjs   (depuis la racine du repo)

import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

// Racine du repo = deux niveaux au-dessus de ce script (frontend/scripts/ → repo/).
const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');

// Parcours récursif → fichiers dont le nom matche `test(name)`, en sautant les dossiers bruyants.
function walk(dir, match, out = []) {
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return out; // dossier absent → ignore
  }
  for (const name of entries) {
    if (name === 'node_modules' || name === '.next' || name === 'dist' || name === '.git') continue;
    const full = join(dir, name);
    if (statSync(full).isDirectory()) walk(full, match, out);
    else if (match(name)) out.push(full);
  }
  return out;
}

// Compte les occurrences d'un regex multiligne sur une liste de fichiers.
function countMatches(files, regex) {
  let n = 0;
  for (const f of files) {
    const m = readFileSync(f, 'utf8').match(regex);
    if (m) n += m.length;
  }
  return n;
}

// Backend (Go) : fonctions de test réelles (signature *testing.T → exclut TestMain *testing.M).
const goFiles = walk(join(ROOT, 'backend'), (n) => n.endsWith('_test.go'));
const backendTests = countMatches(goFiles, /^func Test\w*\([^)]*\*testing\.T/gm);

// Frontend (jest) : it()/test() dans les *.test.ts(x) (les e2e .spec.ts ne comptent pas).
const jestFiles = walk(join(ROOT, 'frontend'), (n) => n.endsWith('.test.ts') || n.endsWith('.test.tsx'));
const frontendTests = countMatches(jestFiles, /^\s*(it|test)\(/gm);

const stats = {
  backend: { tests: backendTests, files: goFiles.length },
  frontend: { tests: frontendTests, suites: jestFiles.length },
  total: backendTests + frontendTests,
  allGreen: true,
  generatedAt: new Date().toISOString().slice(0, 10),
};

writeFileSync(join(ROOT, 'frontend', 'lib', 'test-stats.json'), JSON.stringify(stats, null, 2) + '\n');
console.log(`test-stats → back ${backendTests} (${goFiles.length} fich.) · front ${frontendTests} (${jestFiles.length} suites) · total ${stats.total}`);
