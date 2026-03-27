// Système de hints modulaire pour le LeftPanel.
// Pool de questions par catégorie — affiché sous forme de boutons cliquables.

export type HintCategory = 'projets' | 'skills' | 'freelance' | 'curiosite' | 'blog' | 'meta';

export interface Hint {
  text: string;      // label court affiché sur le bouton
  message: string;   // message envoyé dans le chat au clic
  category: HintCategory;
}

// Pool complet des hints disponibles, alternés par catégorie pour la diversité
export const HINTS: Hint[] = [
  // projets
  { text: "C'est quoi Aria ?", message: "Parle-moi du projet Aria", category: 'projets' },
  { text: "Cogesco c'est quoi ?", message: "Qu'est-ce que le projet Cogesco ?", category: 'projets' },
  { text: "Maicivy, comment ça marche ?", message: "Comment fonctionne maicivy ?", category: 'projets' },
  { text: "Projet le plus complexe ?", message: "Quel est ton projet le plus complexe ?", category: 'projets' },
  // skills
  { text: "Stack IA d'Alexi ?", message: "Quelles sont tes compétences en IA ?", category: 'skills' },
  { text: "Go ou Node ?", message: "Tu aurais choisi Go ou Node pour un nouveau projet ?", category: 'skills' },
  { text: "Tu fais du mobile ?", message: "Est-ce que tu fais du développement mobile ?", category: 'skills' },
  // freelance
  { text: "TJM ?", message: "C'est quoi ton TJM ?", category: 'freelance' },
  { text: "Dispo pour une mission ?", message: "Tu es disponible pour une mission freelance ?", category: 'freelance' },
  { text: "Remote ou présentiel ?", message: "Tu travailles en remote ou présentiel ?", category: 'freelance' },
  // curiosite
  { text: "Galère sur un projet ?", message: "T'as eu des galères techniques sur un projet ?", category: 'curiosite' },
  { text: "Stack idéale ?", message: "C'est quoi ta stack idéale ?", category: 'curiosite' },
  // blog
  { text: "Derniers articles ?", message: "Quels sont tes derniers articles de blog ?", category: 'blog' },
  { text: "Article sur les agents IA ?", message: "T'as écrit sur les agents IA ?", category: 'blog' },
  // meta
  { text: "Comment ce portfolio est fait ?", message: "Comment ce portfolio a été construit ?", category: 'meta' },
  { text: "Qui a codé cette page ?", message: "Qui a développé cette page de chat ?", category: 'meta' },
];

// Tire N hints au hasard depuis le pool.
// Tente d'avoir un hint par catégorie avant d'en répéter — diversité garantie.
export function pickHints(n = 5): Hint[] {
  // Grouper par catégorie
  const byCategory: Record<HintCategory, Hint[]> = {} as Record<HintCategory, Hint[]>;
  for (const hint of HINTS) {
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
