/**
 * tessera.ts — le scénario REJOUÉ « Tessera » : un run d'agents enregistré, sous forme du flux
 * d'événements du contrat (events.ts). Aucun appel IA.
 *
 * QUOI : Inès Morel, CTO (fictive) chez Tessera, décrit un besoin de tri de tickets support ;
 * l'orchestrateur répartit quatre tâches, le Lecteur reformule, le Chercheur lit les fiches et
 * retient sept projets, l'Architecte dessine la chaîne, le Rédacteur écrit cinq sections.
 * POURQUOI un enregistrement au format du flux, et pas une animation scriptée : l'étape suivante
 * remplace CE fichier par un backend Go qui émet les mêmes événements en SSE. Tout ce que l'écran
 * sait faire, il le fait à partir d'événements — le rejeu n'a aucun passe-droit.
 * COMMENT :
 *  - Les instants sont ceux de la maquette validée (maquette.src.html), écrits en secondes de la
 *    maquette et ramenés au clic sur « Lancer l'équipe » par `m()` : chaque horaire se vérifie
 *    contre la source d'un coup d'œil.
 *  - Ce qui est RÉEL n'est pas ici : le nombre de projets indexés, les noms, les chiffres des
 *    preuves (statut, lignes, tests) et les liens de démo viennent des endpoints du site au moment
 *    du rendu (`ReplayContext`, tableau des preuves lu par DesignDossier).
 *  - Les TEXTES des agents (activités, notes, dossier) sont du contenu : ils restent en français,
 *    comme ceux que produiraient les agents pour une cliente francophone. L'interface autour est
 *    traduite (next-intl).
 *  - Les compteurs d'usage sont des ORDRES DE GRANDEUR enregistrés ; le coût enregistré est
 *    calculé ici au barème `RECORDED_RATE` — c'est une donnée de l'enregistrement, pas un tarif que
 *    le front appliquerait : en direct, le coût arrive tout calculé du backend.
 */

import { USAGE_TWEEN } from './runState';
import type { AgentEvent, AgentId, DossierBlock, PipelineEdge, PipelineNode, SectionId, StampedEvent, SubAgentId, ToolFocus, VisitorBrief } from './events';

/** Seconde de la maquette où le visiteur clique sur « Lancer l'équipe » : l'instant 0 du run. */
const LAUNCH = 8.85;
/** Instant maquette → instant du run. */
const m = (s: number) => Math.round((s - LAUNCH) * 1000) / 1000;

/** Données réelles injectées au moment du rendu (endpoints du site). */
export interface ReplayContext {
  /** Nombre de projets de la montagne (GET /api/v1/montagne). */
  projectCount: number;
  /** Démo vidéo de MelodiCode (preuve `video` de sa carte du sommet), URL absolue ; absente = pas de lien. */
  melodicodeVideoUrl?: string;
  /** Le site lui-même, dans la langue du visiteur. */
  siteUrl: string;
}

export interface ReplayScenario {
  visitor: VisitorBrief;
  events: StampedEvent[];
  /** Durée du rejeu (s) : la barre de progression et l'arrêt de l'horloge. */
  duration: number;
}

export const TESSERA_VISITOR: VisitorBrief = {
  name: 'Inès Morel',
  affiliation: 'CTO chez Tessera',
  organisation: 'Tessera',
  initials: 'IM',
  need:
    'Six personnes traitent 1 400 tickets support par jour. Nous voulons des agents IA qui trient chaque ticket, répondent seuls aux cas simples en citant notre base de connaissances, et passent la main à un humain dès que c’est sensible. Avec une trace de chaque décision.',
};

/** Les quatorze candidats retenus par le filtrage de l'index. */
const CANDIDATES = [
  'melodicode', 'maicivy', 'liveconf', 'openclaw_platform', 'envoia', 'aurelm_gm_toolkit', 'momos',
  'claude_orchestrator', 'codeur_notify', 'civjdr_roman', 'demiurgos_game_master', 'anthropic_vendor_proxy',
  'projectmind', 'etheryale_api_gateway',
];

/** Barème de l'ENREGISTREMENT ($ par token, entrée / sortie) — cf. en-tête. */
const RECORDED_RATE = { input: 1e-6, output: 5e-6 };

/**
 * Relevés d'usage enregistrés : [instant maquette, tokens d'entrée, tokens de sortie]. Le
 * Dessinateur n'existait pas à l'enregistrement : le rejeu ne l'annonce pas, il n'a aucun relevé.
 */
type RecordedAgent = Exclude<AgentId, 'diagrammer'>;
const USAGE: Record<RecordedAgent, [number, number, number][]> = {
  orchestrator: [[12.8, 0, 0], [13.8, 1650, 180], [15.0, 2100, 520], [22.4, 3200, 640], [31.8, 4300, 760], [35.6, 5600, 900], [48.4, 6900, 1150], [50.4, 7000, 1210]],
  reader: [[14.5, 0, 0], [15.8, 1300, 90], [19.0, 1500, 330], [22.2, 2140, 610]],
  researcher: [[14.8, 0, 0], [15.7, 3400, 60], [16.8, 3600, 180], [18.4, 5300, 330], [19.8, 6800, 430], [21.3, 8400, 560], [22.8, 10000, 690], [24.1, 11500, 780], [25.6, 13100, 900], [27.1, 14700, 1010], [28.4, 16200, 1080], [29.9, 17800, 1190], [31.4, 19300, 1280], [34.0, 19900, 1420], [35.6, 20100, 1460]],
  architect: [[15.1, 0, 0], [16.6, 2300, 160], [18.4, 4000, 330], [20.2, 5600, 470], [22.6, 6400, 900], [24.4, 7900, 1040], [28.4, 8800, 1560], [30.8, 9600, 1840]],
  writer: [[15.4, 0, 0], [16.2, 1200, 140], [27.0, 3400, 820], [36.3, 6100, 1700], [41.0, 8800, 2600], [44.0, 10100, 3150], [46.4, 11000, 3600], [48.0, 11800, 3900]],
};

/**
 * Relevés toutes les 0,5 s entre deux points enregistrés : le compteur court au lieu de sauter.
 * Chaque relevé part USAGE_TWEEN plus tôt : l'écran fait courir le compteur pendant USAGE_TWEEN
 * après réception (runState.ts), il atteint donc chaque valeur à l'instant de la maquette.
 */
function usageEvents(agent: RecordedAgent): [number, AgentEvent][] {
  const pts = USAGE[agent];
  const out: [number, AgentEvent][] = [];
  const push = (t: number, i: number, o: number) =>
    out.push([t - USAGE_TWEEN, { type: 'usage', agent, inputTokens: Math.round(i), outputTokens: Math.round(o), costUsd: i * RECORDED_RATE.input + o * RECORDED_RATE.output }]);
  for (let k = 1; k < pts.length; k++) {
    const [t0, i0, o0] = pts[k - 1];
    const [t1, i1, o1] = pts[k];
    for (let t = t0 + 0.5; t < t1 - 0.05; t += 0.5) {
      const f = (t - t0) / (t1 - t0);
      push(t, i0 + (i1 - i0) * f, o0 + (o1 - o0) * f);
    }
    push(t1, i1, o1);
  }
  return out;
}

const visitorFocus: ToolFocus = { kind: 'visitor' };
const dossierFocus: ToolFocus = { kind: 'dossier' };
const project = (projectId: string): ToolFocus => ({ kind: 'project', projectId });
const getProject = (id: string) => `get_project("${id}")`;

/** Le schéma d'architecture proposé (section 2), sur une grille colonne × rangée. */
const PIPELINE_NODES: PipelineNode[] = [
  { id: 'ticket', title: 'Ticket', subtitle: 'entrant', col: 0, row: 1 },
  { id: 'triage', title: 'Trieur', subtitle: 'modèle léger', col: 1, row: 1 },
  { id: 'answer', title: 'Répondeur', subtitle: 'cite la base', col: 2, row: 0 },
  { id: 'check', title: 'Contrôle', subtitle: 'sans LLM', col: 3, row: 0 },
  { id: 'human', title: 'File humaine', subtitle: 'brief joint', col: 2, row: 2, human: true },
];
const PIPELINE_EDGES: PipelineEdge[] = [
  { from: 'ticket', to: 'triage', step: 1 },
  { from: 'triage', to: 'answer' },
  { from: 'answer', to: 'check' },
  { from: 'triage', to: 'human', dashed: true, step: 3 },
  { from: 'check', to: 'human', dashed: true, step: 2 },
];

export function tesseraScenario(ctx: ReplayContext): ReplayScenario {
  const ev: [number, AgentEvent][] = [];
  const at = (s: number, e: AgentEvent) => ev.push([m(s), e]);
  const status = (s: number, agent: AgentId, st: 'working' | 'waiting' | 'done', activity?: string) =>
    at(s, { type: 'agent.status', agent, status: st, activity });
  const call = (s: number, agent: AgentId, text: string, focus: ToolFocus) => at(s, { type: 'tool.call', agent, call: text, focus });
  const note = (s: number, agent: SubAgentId, text: string, kind: 'note' | 'question' | 'done' = 'note') =>
    at(s, { type: 'agent.log', agent, kind, text });
  const verdict = (s: number, agent: SubAgentId, projectId: string, v: 'keep' | 'drop', text?: string) => {
    at(s, { type: 'project.verdict', agent, projectId, verdict: v, note: text });
    if (v === 'keep') at(s, { type: 'link.traced', from: 'visitor', to: projectId });
  };
  const say = (s: number, from: AgentId | 'visitor', to: AgentId | 'visitor', text: string, topic?: 'plan' | 'check') =>
    at(s, { type: 'message', from, to, text, topic });

  /* --- Le clic : le besoin part, l'équipe est convoquée. Aucun modèle annoncé : c'est un rejeu. --- */
  at(LAUNCH, {
    type: 'run.started',
    visitor: TESSERA_VISITOR,
    agents: (['orchestrator', 'reader', 'researcher', 'architect', 'writer'] as AgentId[]).map((id) => ({ id })),
  });

  /* --- Orchestrateur --- */
  status(12.8, 'orchestrator', 'working', 'lit la demande');
  say(12.9, 'visitor', 'orchestrator', '« Six personnes traitent 1 400 tickets par jour… »');
  status(13.8, 'orchestrator', 'working', 'répartit le travail');
  say(13.8, 'orchestrator', 'orchestrator', 'quatre tâches indépendantes, en parallèle.', 'plan');
  say(14.0, 'orchestrator', 'reader', 'reformule le besoin, relève ce qui manque.');
  say(14.3, 'orchestrator', 'researcher', 'trouve les projets en production qui le prouvent.');
  say(14.6, 'orchestrator', 'architect', 'propose la chaîne tri, réponse, escalade.');
  say(14.9, 'orchestrator', 'writer', 'monte le dossier, section par section.');
  status(15.0, 'orchestrator', 'working', 'attend les retours');
  say(22.4, 'reader', 'orchestrator', 'besoin reformulé, trois questions ouvertes.');
  say(22.7, 'orchestrator', 'writer', 'le besoin est prêt, première section.');
  say(31.8, 'architect', 'orchestrator', 'chaîne en cinq étapes, trois garde-fous.');
  say(32.0, 'orchestrator', 'writer', 'l’architecture est prête.');
  say(35.6, 'researcher', 'orchestrator', 'sept projets retenus, quatre en preuve.');
  say(35.9, 'orchestrator', 'writer', 'les preuves sont prêtes, chiffres des fiches.');
  status(46.8, 'orchestrator', 'working', 'vérifie le dossier');
  say(46.8, 'orchestrator', 'orchestrator', 'chaque chiffre du dossier contre sa fiche.', 'check');
  say(48.4, 'orchestrator', 'orchestrator', 'lignes, tests et statuts conformes.', 'check');
  status(50.4, 'orchestrator', 'done', 'dossier remis');
  say(50.4, 'orchestrator', 'visitor', 'votre dossier de conception est prêt.');

  /* --- Lecteur du besoin --- */
  status(14.5, 'reader', 'working', 'lit la demande');
  call(14.6, 'reader', 'read_request("ines-morel")', visitorFocus);
  note(15.8, 'reader', '1 400 tickets par jour, six personnes au support');
  note(17.3, 'reader', 'Répondre seulement si une source le justifie');
  note(18.8, 'reader', 'Un humain sur les cas sensibles, décision tracée');
  status(20.0, 'reader', 'working', 'relève ce qui manque');
  call(20.0, 'reader', 'list_gaps(besoin)', visitorFocus);
  note(20.6, 'reader', 'Outil de ticketing non précisé', 'question');
  note(21.0, 'reader', 'Catégories sensibles à définir', 'question');
  note(21.4, 'reader', 'Langues des tickets inconnues', 'question');
  call(21.6, 'reader', 'send("orchestrateur", besoin)', visitorFocus);
  status(22.2, 'reader', 'done');
  note(22.2, 'reader', 'Besoin reformulé, envoyé', 'done');

  /* --- Chercheur de preuves --- */
  status(14.8, 'researcher', 'working', 'lit l’index');
  call(14.9, 'researcher', 'get_projects_index()', visitorFocus);
  note(15.7, 'researcher', `${ctx.projectCount} projets indexés`);
  status(15.9, 'researcher', 'working', 'filtre les projets');
  call(15.9, 'researcher', 'filter(production, agents, humain)', visitorFocus);
  at(15.9, { type: 'index.filtered', agent: 'researcher', total: ctx.projectCount, candidates: CANDIDATES });
  note(16.8, 'researcher', `${CANDIDATES.length} candidats`);
  status(17.2, 'researcher', 'working', 'lit les fiches');
  const reads: [number, number, string, 'keep' | 'drop', string][] = [
    [17.2, 18.4, 'melodicode', 'keep', 'MelodiCode : un coordinateur, des workers, l’humain décide'],
    [18.8, 19.8, 'liveconf', 'drop', 'LiveConf : copilote, aucune délégation'],
    [20.2, 21.3, 'openclaw_platform', 'keep', 'OpenClaw : le client valide les actions'],
    [21.7, 22.8, 'envoia', 'keep', 'EnvoIA : le contenu externe ne décide rien'],
    [23.2, 24.1, 'aurelm_gm_toolkit', 'drop', 'Aurelm : un seul agent'],
    [24.5, 25.6, 'momos', 'keep', 'Momos : explorers parallèles, coût affiché'],
    [26.0, 27.1, 'claude_orchestrator', 'keep', 'ClaudeOrchestrator : workers isolés, disjoncteur'],
    [27.5, 28.4, 'codeur_notify', 'drop', 'Codeur Notify : note les offres, n’agit pas'],
    [28.8, 29.9, 'maicivy', 'keep', 'maicivy : agents en production, démo en ligne'],
    [30.3, 31.4, 'civjdr_roman', 'keep', 'CivJDR Roman : porte de vérification'],
  ];
  for (const [open, decide, id, v, text] of reads) {
    call(open, 'researcher', getProject(id), project(id));
    verdict(decide, 'researcher', id, v, text);
  }
  status(32.0, 'researcher', 'working', 'classe les preuves');
  call(32.0, 'researcher', 'rank(retenus)', visitorFocus);
  // Les candidats jamais ouverts retournent dans le rang (sans note : rien n'a été lu).
  for (const id of ['projectmind', 'etheryale_api_gateway']) verdict(33.0, 'researcher', id, 'drop');
  note(34.0, 'researcher', '4 preuves principales sur 7 retenues');
  call(34.6, 'researcher', 'send("orchestrateur", preuves)', visitorFocus);
  status(35.6, 'researcher', 'done');
  note(35.6, 'researcher', 'Preuves envoyées', 'done');

  /* --- Architecte --- */
  status(15.1, 'architect', 'working', 'lit des architectures');
  call(15.2, 'architect', getProject('melodicode'), project('melodicode'));
  note(16.6, 'architect', 'MelodiCode : un coordinateur délègue, agents isolés');
  call(17.0, 'architect', getProject('envoia'), project('envoia'));
  note(18.4, 'architect', 'EnvoIA : planifier, puis exécuter sans LLM');
  call(18.8, 'architect', getProject('demiurgos_game_master'), project('demiurgos_game_master'));
  verdict(20.2, 'architect', 'demiurgos_game_master', 'keep', 'Demiurgos : un palier de modèle par passe');
  status(20.6, 'architect', 'working', 'dessine la chaîne');
  call(20.6, 'architect', 'draft_pipeline()', visitorFocus);
  note(22.6, 'architect', 'Chaîne : trieur, répondeur, contrôle, humain');
  call(23.0, 'architect', getProject('anthropic_vendor_proxy'), project('anthropic_vendor_proxy'));
  verdict(24.4, 'architect', 'anthropic_vendor_proxy', 'keep', 'Vendor Proxy : bascule si l’API tombe');
  status(24.8, 'architect', 'working', 'pose les garde-fous');
  call(24.8, 'architect', 'define_guardrails()', visitorFocus);
  note(26.0, 'architect', 'Le ticket est cité, jamais exécuté');
  note(27.2, 'architect', 'Contrôle sans LLM avant envoi');
  note(28.4, 'architect', 'Modèle plus fort seulement si doute');
  status(29.2, 'architect', 'working', 'vérifie le besoin');
  call(29.2, 'architect', 'check_against(besoin)', visitorFocus);
  call(30.3, 'architect', 'send("orchestrateur", architecture)', visitorFocus);
  note(30.4, 'architect', 'Couvre les trois exigences');
  status(30.8, 'architect', 'done');
  note(30.8, 'architect', 'Architecture envoyée', 'done');

  /* --- Rédacteur et dossier --- */
  status(15.4, 'writer', 'working', 'ouvre le dossier');
  call(15.5, 'writer', 'create_dossier("Tessera")', dossierFocus);
  at(15.6, {
    type: 'dossier.opened',
    sections: [
      { id: 'need', title: 'Le besoin, reformulé', waitingFor: 'reader' },
      { id: 'architecture', title: 'L’architecture proposée', waitingFor: 'architect' },
      { id: 'proofs', title: 'Les projets qui le prouvent', waitingFor: 'researcher' },
      { id: 'contribution', title: 'Ce qu’Alexis apporte', waitingFor: 'researcher' },
      { id: 'next', title: 'Prochaine étape', waitingFor: 'previous' },
    ],
  });
  status(16.2, 'writer', 'waiting', 'attend le besoin');
  note(16.2, 'writer', 'Plan du dossier : cinq sections');

  const block = (s: number, section: SectionId, b: DossierBlock) => at(s, { type: 'dossier.block', section, block: b });
  const sectionEv = (section: SectionId, s: number, st: 'writing' | 'done') => at(s, { type: 'dossier.section', section, status: st });

  status(22.9, 'writer', 'working', 'écrit le besoin');
  call(22.9, 'writer', 'write_section(1, besoin)', dossierFocus);
  sectionEv('need', 22.9, 'writing');
  block(23.1, 'need', { type: 'lead', text: 'Traiter 1 400 tickets par jour à six, sans perdre la main sur les cas sensibles.' });
  block(25.0, 'need', { type: 'item', text: 'Une réponse automatique seulement si une source de votre base la justifie.' });
  block(25.6, 'need', { type: 'item', text: 'Un passage à un humain sur règle explicite, ticket et contexte joints.' });
  block(26.2, 'need', { type: 'item', text: 'Une trace lisible de chaque décision.' });
  sectionEv('need', 27.0, 'done');
  note(27.0, 'writer', 'Section 1 écrite', 'done');
  status(27.2, 'writer', 'waiting', 'attend l’architecture');

  status(32.5, 'writer', 'working', 'écrit l’architecture');
  call(32.5, 'writer', 'write_section(2, architecture)', dossierFocus);
  sectionEv('architecture', 32.5, 'writing');
  block(32.6, 'architecture', {
    type: 'pipeline',
    nodes: PIPELINE_NODES,
    edges: PIPELINE_EDGES,
    journal: 'Journal des décisions, une ligne par ticket',
    label: 'Chaîne proposée : ticket, trieur, répondeur, contrôle, envoi, avec une file humaine et un journal des décisions',
  });
  block(34.8, 'architecture', { type: 'item', text: 'Le ticket est cité, jamais exécuté.' });
  block(35.3, 'architecture', { type: 'item', text: 'Un contrôle sans LLM avant tout envoi.' });
  block(35.8, 'architecture', { type: 'item', text: 'Un modèle plus fort seulement si le trieur doute.' });
  sectionEv('architecture', 36.3, 'done');
  note(36.3, 'writer', 'Section 2 écrite', 'done');

  status(36.4, 'writer', 'working', 'écrit les preuves');
  call(36.4, 'writer', 'write_section(3, preuves)', dossierFocus);
  sectionEv('proofs', 36.4, 'writing');
  block(37.4, 'proofs', {
    type: 'proof',
    projectId: 'melodicode',
    why: 'Un coordinateur délègue à des agents isolés ; l’humain décide.',
    demo: ctx.melodicodeVideoUrl ? { kind: 'video', url: ctx.melodicodeVideoUrl } : undefined,
  });
  block(38.4, 'proofs', { type: 'proof', projectId: 'openclaw_platform', why: 'SaaS d’agents : le client valide les actions sensibles.' });
  block(39.4, 'proofs', { type: 'proof', projectId: 'envoia', why: 'Un contenu externe ne peut pas infléchir l’action.' });
  block(40.4, 'proofs', { type: 'proof', projectId: 'maicivy', why: 'Ce site : sept outils d’administration à agents.', demo: { kind: 'site', url: ctx.siteUrl } });
  block(40.8, 'proofs', { type: 'also', projectIds: ['momos', 'claude_orchestrator', 'civjdr_roman', 'demiurgos_game_master', 'anthropic_vendor_proxy'] });
  sectionEv('proofs', 41.0, 'done');
  note(41.0, 'writer', 'Section 3 écrite', 'done');

  status(41.2, 'writer', 'working', 'écrit l’apport');
  call(41.2, 'writer', 'write_section(4, apport)', dossierFocus);
  sectionEv('contribution', 41.2, 'writing');
  block(41.5, 'contribution', { type: 'item', text: 'L’architecture et sa mise en production, de bout en bout.' });
  block(42.4, 'contribution', { type: 'item', text: 'Des garde-fous écrits en code et testés comme le reste.' });
  block(43.3, 'contribution', { type: 'item', text: 'Un coût par ticket mesuré, tenu par des paliers de modèles.' });
  sectionEv('contribution', 44.0, 'done');
  note(44.0, 'writer', 'Section 4 écrite', 'done');

  status(44.2, 'writer', 'working', 'écrit la suite');
  call(44.2, 'writer', 'write_section(5, suite)', dossierFocus);
  sectionEv('next', 44.2, 'writing');
  block(44.4, 'next', {
    type: 'paragraph',
    text: 'Un appel de 30 minutes pour trancher l’outil de ticketing, les catégories sensibles et les langues. Puis un prototype sur un échantillon de vos tickets.',
  });
  block(46.0, 'next', { type: 'contact' });
  sectionEv('next', 46.4, 'done');
  note(46.4, 'writer', 'Section 5 écrite', 'done');
  status(46.6, 'writer', 'working', 'compose le PDF');
  call(46.6, 'writer', 'render_pdf(dossier)', dossierFocus);
  status(48.0, 'writer', 'done');
  note(48.0, 'writer', 'PDF composé', 'done');

  at(50.4, { type: 'run.completed' });

  /* --- Usage --- */
  for (const id of Object.keys(USAGE) as RecordedAgent[]) for (const [t, e] of usageEvents(id)) at(t, e);

  // Tri STABLE par instant : à instant égal, l'ordre d'écriture ci-dessus fait foi (un statut
  // « terminé » avant la note « envoyé », un verdict avant son filet).
  const events = ev
    .map(([t, event], i) => ({ t, i, event }))
    .sort((a, b) => a.t - b.t || a.i - b.i)
    .map(({ t, event }) => ({ at: t, event }));
  return { visitor: TESSERA_VISITOR, events, duration: m(52) };
}
