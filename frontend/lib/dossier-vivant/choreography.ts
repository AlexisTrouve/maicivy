/**
 * choreography.ts — les durées et décalages des mouvements d'INTERFACE du dossier vivant.
 *
 * QUOI : ce que fait l'écran en réponse au flux, et combien de temps ça prend — le pliage du champ
 * en tuile, la montagne qui glisse, les paquets dans l'éventail, les pings, le dossier qui monte.
 * POURQUOI séparé du scénario : ces valeurs appartiennent à l'écran, pas aux agents. Un vrai run
 * (SSE) les déclenche de la même façon ; seul l'instant des événements change.
 * Toutes les valeurs sont reprises de la maquette validée (maquette.src.html › T, PACKET_D…),
 * ramenées à l'instant du clic sur « Lancer l'équipe » (8,85 s dans la maquette).
 */
export const CH = {
  /** Après `run.started` (s). */
  fold: 0.45, // le texte d'ouverture sort
  heroOut: 1.0,
  move: 0.55, // la montagne glisse à gauche
  moveD: 3.0,
  open: 1.55, // ses rangs s'ouvrent pour la tuile du visiteur
  openD: 1.6,
  ghostA: 0.65, // le champ se replie en tuile…
  ghostB: 3.35, // … qui se pose
  team: 2.15, // l'équipe entre
  teamD: 1.4,
  dim: 3.55, // les filets réels s'effacent
  dimD: 1.0,

  /** Mouvements déclenchés par un événement (durées, s). */
  sweepD: 0.9, // `index.filtered` : la ligne de balayage
  quietD: 1.0, // puis le reste de la montagne recule
  dropD: 0.6, // un candidat écarté retourne dans le rang
  packetD: 0.5, // un paquet dans l'éventail (message agent ↔ orchestrateur)
  pingD: 1.1, // un anneau autour d'un projet
  threadD: 0.8, // un filet visiteur → projet se trace
  toolCps: 70, // un appel d'outil s'écrit (caractères / s)
  msgReveal: 0.35,
  logReveal: 0.4,
  cursorMove: 0.45,
  cursorFade: 0.3,
  cursorHide: 0.4, // le curseur s'efface peu après « terminé »
  dossierIn: 0.6, // `dossier.opened`
  slideDelay: 0.4, // les sous-agents ont fini → la feuille monte…
  slideD: 1.4,
  fullD: 1.2, // `run.completed` → la feuille prend toute la colonne
  scrollD: 0.8, // la feuille suit la section en cours d'écriture
  blockReveal: 0.45,
  pipelineD: 2.0, // le schéma se dessine élément par élément
  /** Vitesse de frappe des paragraphes du dossier : bornée pour rester lisible. */
  typeCps: 60,
  typeMin: 0.8,
  typeMax: 1.6,
} as const;

/** Durée de frappe d'un paragraphe du dossier. */
export function typingDuration(text: string): number {
  return Math.min(CH.typeMax, Math.max(CH.typeMin, text.length / CH.typeCps));
}
