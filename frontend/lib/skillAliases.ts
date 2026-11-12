// Matching skill ↔ technologies de projet ↔ langage Gitea.
//
// QUOI : fonctions pures qui relient un skill (ex: "Node.js") aux projets qui l'utilisent et à ses
// LOC par langage. C'est le cœur de la fiche détail d'un skill.
// POURQUOI : les noms ne coïncident jamais à l'octet près d'une source à l'autre — "Node.js" (skill)
// vs "JavaScript" (langage Gitea), "Postgres" vs "PostgreSQL", "golang" vs "Go". Sans normalisation,
// un skill afficherait "0 projet" alors qu'il y en a 5 → la feature *aurait l'air* cassée. Ce module
// centralise la canonicalisation + la table d'alias pour que le matching soit fiable et testable.
// COMMENT : 1. `normalizeKey` met en forme (lowercase, trim, retire séparateurs . _ - et espaces) ;
// 2. `canonical` applique la table d'alias par-dessus ; 3. les helpers comparent des clés canoniques.

import { Skill, Project, Experience, LangStat, LangStatsResponse } from './types';

// ALIASES : synonymes → clé canonique. Couvre les écarts connus entre noms de skills, technos de
// projet et langages Gitea. Toute clé est déjà passée par normalizeKey (donc sans séparateurs).
// Ajouter ici dès qu'un skill remonte "0 projet" à tort.
const ALIASES: Record<string, string> = {
  // JavaScript / Node
  nodejs: 'javascript',
  node: 'javascript',
  js: 'javascript',
  javascript: 'javascript',
  // TypeScript
  ts: 'typescript',
  typescript: 'typescript',
  // Go
  golang: 'go',
  go: 'go',
  // Python
  py: 'python',
  python: 'python',
  // C / C++ / C#
  cpp: 'c++',
  'c++': 'c++',
  cplusplus: 'c++',
  csharp: 'c#',
  'c#': 'c#',
  // Postgres
  postgres: 'postgresql',
  postgresql: 'postgresql',
  psql: 'postgresql',
  // React (framework, pas un langage — utile pour le matching projets)
  reactjs: 'react',
  react: 'react',
  // Shell
  bash: 'shell',
  sh: 'shell',
  shell: 'shell',
};

// normalizeKey : forme comparable d'un nom (lowercase, trim, sans séparateurs . _ - ni espaces).
// On conserve volontairement '+' et '#' (significatifs : c++, c#).
export function normalizeKey(raw: string): string {
  return raw.toLowerCase().trim().replace(/[\s._-]+/g, '');
}

// canonical : clé canonique d'un nom (normalisation + alias). Deux noms équivalents (ex: "Node.js"
// et "JavaScript") doivent renvoyer la même valeur.
export function canonical(raw: string): string {
  const k = normalizeKey(raw);
  return ALIASES[k] ?? k;
}

// skillTokens : décompose un nom de skill COMPOSITE en clés canoniques. Beaucoup de skills réels
// regroupent plusieurs langages ("C/C++", "Flutter / Dart") → un seul token raterait les LOC du
// 2e (Dart = 300k+ lignes !). On découpe sur / et , UNIQUEMENT — pas sur l'espace ni le point,
// pour ne pas casser "Node.js" ou "Visual Basic". Chaque morceau passe par canonical (alias inclus).
export function skillTokens(raw: string): string[] {
  return raw
    .split(/[/,]/)
    .map((s) => canonical(s))
    .filter(Boolean);
}

// projectsForSkill : projets dont une techno, le langage principal, OU un tag (flags-concept inclus)
// matche un des tokens du skill. Les skills-concept (Scraping, AI/LLM…) matchent via les tags ;
// les langages/outils via technologies — cf. [[décision]] : tags portés au frontend pour le matching.
export function projectsForSkill(skill: Skill, projects: Project[] = []): Project[] {
  const keys = new Set(skillTokens(skill.name));
  return projects.filter((p) => {
    const techKeys = (p.technologies ?? []).map(canonical);
    if (p.language) techKeys.push(canonical(p.language));
    // Les tags portent les flags-concept ("AI / LLM Integration", "MCP (Model Context Protocol)"…).
    // On les tokenise comme les noms de skill (skillTokens) pour que le split sur '/' soit symétrique
    // des deux côtés — sinon canonical("AI / LLM Integration") garderait le '/' et ne matcherait jamais.
    (p.tags ?? []).forEach((t) => skillTokens(t).forEach((tok) => techKeys.push(tok)));
    return techKeys.some((t) => keys.has(t));
  });
}

// experiencesForSkill : expériences dont une techno matche un des tokens du skill.
export function experiencesForSkill(skill: Skill, experiences: Experience[] = []): Experience[] {
  const keys = new Set(skillTokens(skill.name));
  return experiences.filter((e) => (e.technologies ?? []).map(canonical).some((t) => keys.has(t)));
}

// locForSkill : LOC agrégées d'un skill, SI au moins un de ses tokens correspond à un langage du map
// Gitea. Somme les tokens matchés (ex: "C/C++" = c + c++). Retourne null pour les skills non-langages
// (React, Docker, Excel…) → la fiche masque le bloc LOC (adaptatif "en fonction de la situation").
export function locForSkill(skill: Skill, langStats?: LangStatsResponse | null): LangStat | null {
  if (!langStats?.languages) return null;
  const keys = new Set(skillTokens(skill.name));
  let bytes = 0;
  let loc = 0;
  let matched = false;
  for (const [lang, stat] of Object.entries(langStats.languages)) {
    if (keys.has(canonical(lang))) {
      bytes += stat.bytes;
      loc += stat.loc;
      matched = true;
    }
  }
  return matched ? { language: canonical(skill.name), bytes, loc } : null;
}
