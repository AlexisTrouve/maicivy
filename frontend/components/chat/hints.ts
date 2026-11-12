// Système de hints pour le LeftPanel.
// QUOI : sélection diversifiée de questions-suggestions affichées en boutons cliquables.
// POURQUOI i18n : les LIBELLÉS (text + message) sont dans les fichiers de traduction
//   (messages/*.json → `chat.hints`), lus via `t.raw('hints')` côté composant. On ne hardcode
//   AUCUNE phrase ici. Ce fichier ne garde que le TYPE et la logique de tirage (indépendante de la langue).

export type HintCategory = 'projets' | 'skills' | 'freelance' | 'curiosite' | 'blog' | 'meta';

export interface Hint {
  text: string;      // label court affiché sur le bouton
  message: string;   // message envoyé dans le chat au clic
  category: HintCategory;
}

// Tire N hints au hasard depuis le pool fourni (déjà localisé par l'appelant).
// Tente d'avoir un hint par catégorie avant d'en répéter — diversité garantie.
export function pickHints(hints: Hint[], n = 5): Hint[] {
  // Grouper par catégorie
  const byCategory: Record<HintCategory, Hint[]> = {} as Record<HintCategory, Hint[]>;
  for (const hint of hints) {
    if (!byCategory[hint.category]) byCategory[hint.category] = [];
    byCategory[hint.category].push(hint);
  }

  // Shuffle chaque catégorie indépendamment
  const shuffled: Record<HintCategory, Hint[]> = {} as Record<HintCategory, Hint[]>;
  for (const cat of Object.keys(byCategory) as HintCategory[]) {
    shuffled[cat] = [...byCategory[cat]].sort(() => Math.random() - 0.5);
  }

  // Alterner entre catégories : prend un hint par catégorie en boucle
  const categories = (Object.keys(shuffled) as HintCategory[]).sort(() => Math.random() - 0.5);
  const result: Hint[] = [];
  let round = 0;

  while (result.length < n) {
    let added = false;
    for (const cat of categories) {
      if (result.length >= n) break;
      const idx = Math.floor(round / categories.length);
      const pool = shuffled[cat];
      if (idx < pool.length) {
        result.push(pool[idx]);
        added = true;
      }
    }
    round += categories.length;
    // Sécurité : si plus rien à piocher, arrêter
    if (!added) break;
  }

  return result.slice(0, n);
}
