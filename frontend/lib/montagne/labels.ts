/**
 * labels.ts — quelles étiquettes s'affichent, image par image. Pur calcul, aucun DOM.
 *
 * QUOI : chaque nœud a toujours son icône ; son étiquette (le nom) s'affiche si elle a la place,
 * sinon elle s'estompe. Rend l'ensemble des ids dont l'étiquette est affichée.
 *
 * POURQUOI : décision d'Alexis — « icône + nom toujours affichés ». Mesuré sur la première version :
 * 75 étiquettes dessinées sans arbitrage se chevauchaient par dizaines en vue d'ensemble (« Confluent »
 * sur « WeChat Homework Bot »), et l'œil ne pouvait plus en lire aucune. Une étiquette qui en
 * chevaucherait une autre cède donc la place ; l'icône, elle, ne disparaît jamais, et l'étiquette
 * revient dès que la place se libère (zoom, rotation) — le calcul est refait à chaque image.
 *
 * QUI GAGNE — ordre déterministe, du plus prioritaire au moins prioritaire :
 *  1. le nœud survolé ou ouvert (`focus`) : l'utilisateur le désigne, il doit le lire ;
 *  2. le plus PROCHE de la caméra (profondeur croissante) : il est devant, se peint par-dessus les
 *     autres et paraît le plus grand — c'est lui que l'œil associe à l'étiquette. Tourner la
 *     montagne amène donc d'autres nœuds devant, et leurs étiquettes avec eux ;
 *  3. à profondeur égale, le plus relié (degré décroissant) : un carrefour du réseau explique plus
 *     de liens ;
 *  4. l'id, pour qu'aucune égalité ne dépende de l'ordre des données.
 *
 * COMMENT : glouton dans cet ordre. Une étiquette est acceptée si elle tient ENTIÈRE dans le cadre
 * (une étiquette coupée au bord ne se lit pas), ne touche aucune étiquette déjà acceptée, et ne
 * recouvre aucune icône autre que la sienne (l'icône doit rester visible). Une marge `gap` sépare
 * deux étiquettes voisines : collées, deux noms se lisent comme un seul.
 */

export interface Rect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

export interface LabelCandidate {
  id: string;
  /** Rectangle de l'étiquette (px de la scène). */
  label: Rect;
  /** Rectangle de l'icône du même nœud : ignoré comme obstacle pour sa propre étiquette. */
  icon: Rect;
  depth: number;
  degree: number;
  focus: boolean;
}

/** Vrai si deux rectangles, chacun élargi de `gap / 2`, se recouvrent. */
export function intersects(a: Rect, b: Rect, gap = 0): boolean {
  return a.left < b.right + gap && b.left < a.right + gap && a.top < b.bottom + gap && b.top < a.bottom + gap;
}

/** Ordre de priorité (cf. en-tête). Négatif si `a` passe avant `b`. */
export function comparePriority(a: LabelCandidate, b: LabelCandidate): number {
  if (a.focus !== b.focus) return a.focus ? -1 : 1;
  if (a.depth !== b.depth) return a.depth - b.depth;
  if (a.degree !== b.degree) return b.degree - a.degree;
  return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
}

/**
 * Les ids dont l'étiquette s'affiche. `bounds` = cadre où une étiquette doit tenir entière ;
 * `reserved` = rectangles déjà occupés (noms des strates), qu'aucune étiquette ne recouvre.
 * Le nœud désigné (`focus`) est seul dispensé de la règle des icônes : on le pointe pour le lire,
 * et l'icône qu'il couvre réapparaît dès qu'il n'est plus désigné.
 */
export function chooseLabels(candidates: LabelCandidate[], bounds: Rect, reserved: Rect[] = [], gap = 4): Set<string> {
  const ordered = [...candidates].sort(comparePriority);
  const accepted: Rect[] = [...reserved];
  const shown = new Set<string>();
  for (const c of ordered) {
    const l = c.label;
    const inside = l.left >= bounds.left && l.right <= bounds.right && l.top >= bounds.top && l.bottom <= bounds.bottom;
    if (!inside) continue;
    if (accepted.some((r) => intersects(l, r, gap))) continue;
    if (!c.focus && candidates.some((o) => o.id !== c.id && intersects(l, o.icon))) continue;
    accepted.push(l);
    shown.add(c.id);
  }
  return shown;
}

/**
 * Coupe un nom en lignes d'au plus `maxWidth` px, sur les espaces, au plus `maxLines` lignes.
 * POURQUOI : quelques noms sont très longs (« Suite de développement IA pour Unreal Engine »,
 * 44 caractères ≈ 260 px) ; sur une ligne, leur étiquette ne trouverait presque jamais sa place.
 * Sur deux lignes, le nom reste entier. Seul ce qui dépasse encore la dernière ligne est abrégé
 * par « … » — le nom complet reste lisible au survol (nœud désigné) et dans la liste.
 * `measure` = largeur d'un texte dans la police courante (ctx.measureText côté canvas).
 */
export function wrapLabel(name: string, maxWidth: number, measure: (s: string) => number, maxLines = 2): string[] {
  if (measure(name) <= maxWidth) return [name];
  const words = name.split(/\s+/);
  const lines: string[] = [];
  let line = '';
  for (let i = 0; i < words.length; i++) {
    const trial = line ? `${line} ${words[i]}` : words[i];
    if (!line || measure(trial) <= maxWidth) {
      line = trial;
      continue;
    }
    lines.push(line);
    if (lines.length === maxLines - 1) {
      line = words.slice(i).join(' ');
      break;
    }
    line = words[i];
  }
  lines.push(line);
  // Dernière ligne trop longue (ou mot unique trop long) : abrégée, jamais débordante.
  const last = lines.length - 1;
  if (measure(lines[last]) > maxWidth) {
    let cut = lines[last];
    while (cut.length > 1 && measure(`${cut}…`) > maxWidth) cut = cut.slice(0, -1);
    lines[last] = `${cut.trimEnd()}…`;
  }
  return lines;
}
