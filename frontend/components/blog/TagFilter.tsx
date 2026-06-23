'use client';

import React from 'react';

/**
 * QUOI : barre latérale de filtrage des articles de blog par tag.
 *
 * POURQUOI : le blog accumule des tags auto-générés (50+ pour une douzaine d'articles) et
 * l'utilisateur veut restreindre la liste à un tag donné. Le filtrage est 100% côté client
 * (les tags sont déjà présents dans la réponse de liste — pas de content/content_html), donc
 * aucun appel réseau supplémentaire : on filtre les posts déjà chargés par le parent.
 *
 * COMMENT : le composant est "bête" — il reçoit la liste des tags DÉJÀ comptés et triés
 * (le calcul fréquence vit dans le parent, source unique des posts), le tag sélectionné, et
 * un callback. Un clic sur un tag actif (ou sur "Tous") remet le filtre à zéro. Les
 * data-testid sont stables pour le test E2E (la doctrine : une UI cliquable sans test qui
 * clique pour de vrai = non vérifiée).
 */

export interface TagCount {
  tag: string; // valeur filtrée (clé stable, ex: "games") — aussi utilisée pour le data-testid
  count: number;
  label?: string; // libellé affiché (ex: "Jeux") ; défaut = la valeur `tag`
}

interface TagFilterProps {
  tags: TagCount[];
  selected: string | null;
  total: number;
  locale?: string;
  onSelect: (tag: string | null) => void;
}

export function TagFilter({ tags, selected, total, locale = 'fr', onSelect }: TagFilterProps) {
  // Styles partagés : un bouton = ligne pleine largeur, label à gauche + compteur à droite.
  const baseBtn =
    'w-full text-left px-3 py-1.5 rounded-lg text-sm transition-colors flex items-center justify-between gap-2';
  const activeBtn = 'bg-blue-600 text-white';
  const idleBtn = 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700';

  return (
    <aside data-testid="tag-filter" className="lg:w-56 lg:shrink-0">
      {/* sticky : le filtre reste visible quand on scrolle la grille d'articles (desktop). */}
      <div className="lg:sticky lg:top-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-3">
          {locale === 'fr' ? 'Filtrer par thème' : 'Filter by theme'}
        </h2>

        {/* Liste scrollable : 50+ tags ne doivent pas pousser la page. Hauteur compacte en
            mobile (le filtre est au-dessus du contenu), plus généreuse en desktop. */}
        <div className="max-h-64 lg:max-h-[70vh] overflow-y-auto pr-1 space-y-1">
          {/* "Tous" — réinitialise le filtre (selected === null). */}
          <button
            type="button"
            data-testid="tag-all"
            onClick={() => onSelect(null)}
            className={`${baseBtn} ${selected === null ? activeBtn : idleBtn}`}
          >
            <span>{locale === 'fr' ? 'Tous' : 'All'}</span>
            <span className={selected === null ? 'opacity-80' : 'text-gray-400'}>{total}</span>
          </button>

          {/* Un bouton par tag (déjà triés par fréquence décroissante côté parent). Cliquer
              le tag déjà actif le désélectionne (retour à "Tous"). */}
          {tags.map(({ tag, count, label }) => {
            const isActive = selected === tag;
            return (
              <button
                key={tag}
                type="button"
                data-testid={`tag-${tag}`}
                onClick={() => onSelect(isActive ? null : tag)}
                className={`${baseBtn} ${isActive ? activeBtn : idleBtn}`}
              >
                <span className="truncate">{label ?? tag}</span>
                <span className={isActive ? 'opacity-80' : 'text-gray-400'}>{count}</span>
              </button>
            );
          })}
        </div>
      </div>
    </aside>
  );
}
