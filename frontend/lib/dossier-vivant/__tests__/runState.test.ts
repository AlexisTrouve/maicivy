/**
 * Le repli du flux (runState.ts) sur le VRAI scénario rejoué (tessera.ts) : ce que l'écran doit
 * montrer à des instants clés du parcours validé dans la maquette.
 */
import { SECTION_IDS, type StampedEvent } from '../events';
import { projectLook } from '../looks';
import { elapsedAt, foldEvents, researchDoneAt, totalsAt, usageAt } from '../runState';
import { tesseraScenario } from '../tessera';

const scenario = tesseraScenario({ projectCount: 75, siteUrl: '/fr', melodicodeVideoUrl: 'https://x/v' });
/** État à l'instant `t` du run, comme le rendrait ReplaySource. */
const at = (t: number) => foldEvents(scenario.events.filter((e) => e.at <= t));
/** Instant maquette → instant du run (clic sur « Lancer » à 8,85 s). */
const m = (s: number) => s - 8.85;

describe('tesseraScenario', () => {
  it('est trié par instant et commence par run.started à 0', () => {
    const times = scenario.events.map((e) => e.at);
    expect([...times].sort((a, b) => a - b)).toEqual(times);
    expect(scenario.events[0]).toMatchObject({ at: 0, event: { type: 'run.started' } });
  });

  it("n'annonce aucun modèle : le rejeu ne prétend à aucun", () => {
    const started = scenario.events[0].event as Extract<StampedEvent['event'], { type: 'run.started' }>;
    expect(started.agents.every((a) => a.model === undefined)).toBe(true);
  });

  it('lit le nombre de projets dans le contexte réel', () => {
    const s = at(m(16));
    expect(s.agents.researcher.log.map((l) => l.text)).toContain('75 projets indexés');
  });
});

describe('foldEvents', () => {
  it('à 24 s : les quatre sous-agents travaillent, trois projets sont déjà allumés', () => {
    const s = at(m(24));
    expect(['reader', 'researcher', 'architect', 'writer'].map((id) => s.agents[id as 'reader'].status)).toEqual(['done', 'working', 'working', 'working']);
    const lit = [...s.verdicts].filter(([, v]) => v.verdict === 'keep').map(([id]) => id);
    expect(lit.sort()).toEqual(['demiurgos_game_master', 'envoia', 'melodicode', 'openclaw_platform']);
    expect(s.dossier?.sections.need.status).toBe('writing');
  });

  it('un get_project met le projet en lecture, la note suivante le libère', () => {
    expect(at(m(17.5)).agents.researcher.reading).toBe('melodicode');
    expect(at(m(18.5)).agents.researcher.reading).toBeNull();
  });

  it('un statut « terminé » vide la ligne d’appel d’outil', () => {
    expect(at(m(22)).agents.reader.call?.text).toBe('send("orchestrateur", besoin)');
    expect(at(m(22.3)).agents.reader.call).toBeNull();
  });

  it('à la remise : neuf projets allumés et tracés (7 du Chercheur, 2 de l’Architecte), cinq sections écrites, revue faite', () => {
    const s = at(scenario.duration);
    const kept = [...s.verdicts].filter(([, v]) => v.verdict === 'keep');
    expect(kept).toHaveLength(9);
    expect(s.threads.size).toBe(9);
    expect(SECTION_IDS.every((id) => s.dossier!.sections[id].status === 'done')).toBe(true);
    expect(s.dossier!.sections.proofs.blocks.filter((b) => b.block.type === 'proof')).toHaveLength(4);
    expect(s.reviewAt).not.toBeNull();
    expect(s.completedAt).toBeCloseTo(m(50.4));
  });

  it('les candidats jamais lus retournent dans le rang', () => {
    const s = at(m(34));
    expect(s.verdicts.get('projectmind')?.verdict).toBe('drop');
    expect(projectLook(s, 'projectmind', m(34), 0).look).toBe('dropped');
  });

  it('le dossier monte quand Lecteur, Chercheur et Architecte ont fini', () => {
    expect(researchDoneAt(at(m(35)))).toBeNull();
    expect(researchDoneAt(at(m(36)))).toBeCloseTo(m(35.6));
  });
});

describe('sélecteurs continus', () => {
  it('le compteur court entre deux relevés au lieu de sauter', () => {
    const s = at(m(20.1));
    const a = usageAt(s, 'researcher', m(20.1)).tokens;
    const b = usageAt(s, 'researcher', m(20.3)).tokens;
    expect(b).toBeGreaterThan(a);
  });

  it('totaux finaux : ceux de la maquette (59 660 tokens, 0,096 $)', () => {
    const end = scenario.duration;
    const s = at(end);
    const tot = totalsAt(s, end);
    expect(tot.tokens).toBe(59660);
    expect(tot.costUsd).toBeCloseTo(0.0961, 3);
  });

  it('le temps de travail court du premier agent à la remise (0:37 dans la maquette)', () => {
    const s = at(scenario.duration);
    expect(elapsedAt(s, scenario.duration)).toBeCloseTo(50.4 - 12.8, 1);
  });
});

describe('run.failed (run réel arrêté)', () => {
  it('garde le PREMIER arrêt nommé, sans toucher à ce qui a été montré', () => {
    const before = at(m(20));
    const failed = foldEvents(
      [
        { at: 12, event: { type: 'run.failed', reason: 'model' } },
        { at: 13, event: { type: 'run.failed', reason: 'network' } }, // la source ajoute la coupure ensuite
      ],
      before,
    );
    expect(failed.failure).toEqual({ at: 12, reason: 'model' });
    expect(failed.completedAt).toBeNull();
    expect(failed.verdicts).toBe(before.verdicts);
  });
});

describe('équipe du run (Dessinateur)', () => {
  const visitor = { name: 'C', affiliation: 'A', organisation: 'O', initials: 'CO', need: 'n' };
  const startedWith = (ids: string[]): StampedEvent => ({
    at: 0,
    event: { type: 'run.started', visitor, agents: ids.map((id) => ({ id: id as 'reader', model: id === 'diagrammer' ? 'claude-sonnet-5-5' : 'claude-haiku-5-5' })) },
  });

  it('le rejeu n’annonce pas le Dessinateur : quatre colonnes', () => {
    expect(at(0).team).toEqual(['reader', 'researcher', 'architect', 'writer']);
  });

  it('un run réel l’annonce : cinq colonnes, dans l’ordre du contrat, chacune avec SON modèle', () => {
    const s = foldEvents([startedWith(['orchestrator', 'reader', 'researcher', 'architect', 'diagrammer', 'writer'])]);
    expect(s.team).toEqual(['reader', 'researcher', 'architect', 'diagrammer', 'writer']);
    expect(s.agents.diagrammer.model).toBe('claude-sonnet-5-5');
    expect(s.agents.architect.model).toBe('claude-haiku-5-5');
  });

  it('son usage entre dans les totaux de l’équipe', () => {
    const s = foldEvents([
      startedWith(['orchestrator', 'reader', 'researcher', 'architect', 'diagrammer', 'writer']),
      { at: 1, event: { type: 'usage', agent: 'architect', inputTokens: 100, outputTokens: 50, costUsd: 0.001 } },
      { at: 2, event: { type: 'usage', agent: 'diagrammer', inputTokens: 2000, outputTokens: 3000, costUsd: 0.051 } },
    ]);
    expect(totalsAt(s, 10)).toEqual({ tokens: 5150, costUsd: 0.052 });
  });
});

describe('identité des messages', () => {
  it('deux messages du même paquet réseau (même instant, mêmes interlocuteurs) restent deux messages distincts', () => {
    // Cas réel (schema-run.ndjson) : le plan puis le contrôle de l'orchestrateur, reçus dans le même
    // paquet SSE, donc horodatés au même instant — leur clé React était identique.
    const s = foldEvents([
      { at: 0.032, event: { type: 'message', from: 'orchestrator', to: 'orchestrator', text: 'plan', topic: 'plan' } },
      { at: 0.032, event: { type: 'message', from: 'orchestrator', to: 'orchestrator', text: 'contrôle', topic: 'check' } },
    ]);
    expect(s.messages.map((m) => m.id)).toEqual([0, 1]);
  });

  it('sur tout le rejeu Tessera, chaque message a un rang unique', () => {
    const ids = at(scenario.duration).messages.map((m) => m.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
