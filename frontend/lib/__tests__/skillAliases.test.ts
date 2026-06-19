// Tests PURS (aucun import next-intl / composant) — donc immunisés au souci de transform ESM jest.
// Verrouillent le matching skill ↔ projets / langage, qui est le point de fragilité n°1 de la fiche.

import {
  normalizeKey,
  canonical,
  skillTokens,
  projectsForSkill,
  experiencesForSkill,
  locForSkill,
} from '../skillAliases';
import { Skill, Project, Experience, LangStatsResponse } from '../types';

const skill = (name: string, category = 'backend'): Skill => ({
  id: name,
  name,
  level: 'expert',
  category,
  yearsExperience: 3,
});

describe('normalizeKey', () => {
  it('lowercase, trim, retire les séparateurs mais garde + et #', () => {
    expect(normalizeKey('  Node.js ')).toBe('nodejs');
    expect(normalizeKey('C++')).toBe('c++');
    expect(normalizeKey('C#')).toBe('c#');
    expect(normalizeKey('Type-Script')).toBe('typescript');
  });
});

describe('canonical (alias)', () => {
  it('réduit les synonymes à la même clé', () => {
    expect(canonical('Node.js')).toBe('javascript');
    expect(canonical('JavaScript')).toBe('javascript');
    expect(canonical('golang')).toBe('go');
    expect(canonical('Postgres')).toBe('postgresql');
    expect(canonical('PostgreSQL')).toBe('postgresql');
  });
  it('laisse passer les noms inconnus normalisés', () => {
    expect(canonical('Rust')).toBe('rust');
  });
});

describe('skillTokens (noms composites)', () => {
  it('découpe sur / et , et canonicalise chaque morceau', () => {
    expect(skillTokens('C/C++')).toEqual(['c', 'c++']);
    expect(skillTokens('Flutter / Dart')).toEqual(['flutter', 'dart']);
    expect(skillTokens('Node.js')).toEqual(['javascript']); // pas de découpe sur le point
    expect(skillTokens('Go')).toEqual(['go']);
  });
});

describe('projectsForSkill', () => {
  const projects: Project[] = [
    { id: '1', title: 'API', description: '', technologies: ['Go', 'PostgreSQL'], featured: true },
    { id: '2', title: 'Web', description: '', technologies: ['React', 'TypeScript'], language: 'TypeScript', featured: false },
    { id: '3', title: 'CLI', description: '', technologies: ['Node.js'], featured: false },
  ];

  it('matche par technologie', () => {
    const r = projectsForSkill(skill('Go'), projects);
    expect(r.map((p) => p.id)).toEqual(['1']);
  });

  it('matche par langage principal du projet', () => {
    const r = projectsForSkill(skill('TypeScript'), projects);
    expect(r.map((p) => p.id)).toEqual(['2']);
  });

  it('relie "JavaScript" à un projet listant "Node.js" via les alias', () => {
    const r = projectsForSkill(skill('JavaScript'), projects);
    expect(r.map((p) => p.id)).toEqual(['3']);
  });

  it('aucun match → tableau vide (pas de crash)', () => {
    expect(projectsForSkill(skill('Rust'), projects)).toEqual([]);
    expect(projectsForSkill(skill('Go'), [])).toEqual([]);
  });

  it('matche un skill-concept via les tags (flags-concept maiProFiles)', () => {
    const tagged: Project[] = [
      { id: 'a', title: 'Scraper', description: '', technologies: ['Python', 'camoufox'], tags: ['Scraping', 'AI / LLM Integration'], featured: false },
      { id: 'b', title: 'Plain', description: '', technologies: ['Go'], featured: false },
    ];
    // "Scraping" ne matche aucune techno mais matche le tag
    expect(projectsForSkill(skill('Scraping'), tagged).map((p) => p.id)).toEqual(['a']);
    expect(projectsForSkill(skill('AI / LLM Integration'), tagged).map((p) => p.id)).toEqual(['a']);
    // Un skill sans correspondance reste vide
    expect(projectsForSkill(skill('Rust'), tagged)).toEqual([]);
  });
});

describe('experiencesForSkill', () => {
  const exps: Experience[] = [
    { id: 'e1', title: 'Dev', company: 'X', description: '', startDate: '2022-01-01', technologies: ['Go', 'Docker'], tags: [] },
  ];
  it('matche les technologies', () => {
    expect(experiencesForSkill(skill('Docker'), exps).map((e) => e.id)).toEqual(['e1']);
    expect(experiencesForSkill(skill('Rust'), exps)).toEqual([]);
  });
});

describe('locForSkill', () => {
  const langStats: LangStatsResponse = {
    languages: {
      go: { language: 'go', bytes: 380000, loc: 10000 },
      typescript: { language: 'typescript', bytes: 76000, loc: 2000 },
    },
    totalLoc: 12000,
    totalBytes: 456000,
    period: 'all-time',
  };

  it('retourne les LOC pour un skill-langage', () => {
    expect(locForSkill(skill('Go'), langStats)?.loc).toBe(10000);
    expect(locForSkill(skill('golang'), langStats)?.loc).toBe(10000);
  });

  it('retourne null pour un skill non-langage (React, Docker…)', () => {
    expect(locForSkill(skill('React', 'frontend'), langStats)).toBeNull();
    expect(locForSkill(skill('Docker', 'devops'), langStats)).toBeNull();
  });

  it('retourne null si pas de stats', () => {
    expect(locForSkill(skill('Go'), null)).toBeNull();
    expect(locForSkill(skill('Go'), undefined)).toBeNull();
  });

  it('agrège les tokens d’un skill composite ("C/C++" = c + c++)', () => {
    const stats: LangStatsResponse = {
      languages: {
        c: { language: 'c', bytes: 38000, loc: 1000 },
        'c++': { language: 'c++', bytes: 380000, loc: 10000 },
        dart: { language: 'dart', bytes: 380000, loc: 10000 },
      },
      totalLoc: 21000,
      totalBytes: 798000,
      period: 'all-time',
    };
    // "C/C++" doit sommer c (1000) + c++ (10000) = 11000
    expect(locForSkill(skill('C/C++'), stats)?.loc).toBe(11000);
    // "Flutter / Dart" doit récupérer dart (10000) même si "flutter" n’est pas un langage
    expect(locForSkill(skill('Flutter / Dart', 'frontend'), stats)?.loc).toBe(10000);
  });
});
