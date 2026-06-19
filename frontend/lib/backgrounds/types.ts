/**
 * Contrat commun à tous les fonds animés ("plugins") du site.
 *
 * QUOI : définit l'interface qu'un fond doit implémenter pour être branchable dans
 *   <BackgroundHost> (la coquille partagée).
 * POURQUOI : la coquille centralise UNE SEULE FOIS les invariantes communes — skip mobile,
 *   defer post-LCP, prefers-reduced-motion, resize, RAF loop, cleanup GPU. Chaque plugin
 *   n'écrit QUE sa logique de rendu. Sans ce contrat, chaque fond redupliquerait ce
 *   boilerplate et finirait par en oublier une (fuite GPU, crash mobile, etc.) — c'est
 *   exactement ce qui "casse quelque chose" quand on swappe un fond.
 * COMMENT : un plugin exporte une fonction `init(ctx)` qui construit son rendu dans le
 *   conteneur fourni et renvoie des hooks de cycle de vie (frame/resize/onPointerMove/dispose).
 */

// Contexte fourni par la coquille au plugin lors de son init.
export interface BackgroundContext {
  mount: HTMLDivElement; // conteneur fixed inset-0 ; le plugin y append son <canvas>
  width: number; // largeur viewport en px CSS
  height: number; // hauteur viewport en px CSS
  dpr: number; // devicePixelRatio déjà plafonné par la coquille (perf)
  reducedMotion: boolean; // true si l'utilisateur préfère moins d'animation
}

// Hooks de cycle de vie renvoyés par init — tous optionnels sauf dispose.
export interface BackgroundInstance {
  // Appelé par la RAF loop de la coquille à chaque frame. dtMs = ms depuis la frame précédente.
  // Un fond statique peut l'omettre (rendu une seule fois dans init).
  frame?: (dtMs: number) => void;
  // Appelé au resize du viewport — le plugin ajuste caméra/renderer/buffers.
  resize?: (width: number, height: number) => void;
  // Position souris (coords viewport CSS). Optionnel : seuls les fonds interactifs l'implémentent.
  onPointerMove?: (x: number, y: number) => void;
  // Libère TOUTES les ressources (GPU, timers, listeners internes). Appelé au switch/unmount.
  dispose: () => void;
}

// Signature d'init d'un plugin. Async autorisé : un plugin lourd (Three.js) fait son
// import() dynamique ici, ce qui garde sa dépendance hors du bundle initial.
export type BackgroundInitFn = (
  ctx: BackgroundContext
) => BackgroundInstance | Promise<BackgroundInstance>;

// Entrée du manifeste — LÉGÈRE (aucune dép lourde), toujours chargée pour peupler le sélecteur.
// La logique de rendu n'est chargée que via load() quand le fond est sélectionné.
export interface BackgroundManifestEntry {
  id: string; // identifiant stable (URL ?bg=, localStorage, clé de rendu)
  name: string; // nom affiché (technique/propre, non traduit : "Constellation", "Game of Life"...)
  mobile: boolean; // true = autorisé sur <768px ; false = skip mobile (trop coûteux)
  load: () => Promise<{ default: BackgroundInitFn }>; // import() lazy de la logique de rendu
}
