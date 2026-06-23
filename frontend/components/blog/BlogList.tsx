'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { BlogPost } from '@/lib/types';
import { blogApi } from '@/lib/api';
import { BlogCard } from './BlogCard';
import { TagFilter, TagCount } from './TagFilter';

interface BlogListProps {
  locale?: string;
  initialPosts?: BlogPost[];
}

// Plafond de chargement. POURQUOI : le filtre par tag est 100% client → il faut TOUS les
// posts en mémoire pour filtrer correctement (sinon un tag de la page 2 serait invisible).
// maicivy borne per_page à 50 ; le blog est très en-dessous. Si un jour on dépasse 50, il
// faudra paginer/charger en plusieurs fois (ou filtrer côté backend) — voir docs/PUBLISHERS.
const FETCH_ALL_LIMIT = 50;

// Thèmes curated (catégories du filtre). Liste FIXE — on ne filtre PAS sur les ~50 tags
// auto-générés (bruyants, incohérents) mais sur ces catégories posées par le pipeline WanMira.
// COUPLAGE : les `key` doivent correspondre au `theme` déclaré dans les profils WanMira
// (devblog.yaml → "tech", drifterra.yaml / futurs jeux → "games"). Un article appartient à un
// thème si la clé figure dans ses tags (le publisher WanMira y pousse le thème). Ajouter un
// jeu = un profil avec `theme: games` → rien à changer ici.
const THEMES: { key: string; fr: string; en: string }[] = [
  { key: 'tech', fr: 'Tech', en: 'Tech' },
  { key: 'games', fr: 'Jeux', en: 'Games' },
];

export function BlogList({ locale = 'fr', initialPosts }: BlogListProps) {
  const [posts, setPosts] = useState<BlogPost[]>(initialPosts || []);
  const [loading, setLoading] = useState(!initialPosts);
  const [error, setError] = useState<string | null>(null);
  // Tag actif (null = "Tous"). Filtrage purement client sur les posts déjà chargés.
  const [selectedTag, setSelectedTag] = useState<string | null>(null);

  useEffect(() => {
    if (!initialPosts) {
      fetchPosts();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const fetchPosts = async () => {
    try {
      setLoading(true);
      setError(null);
      // Une seule requête, tous les posts (tri date décroissante déjà fait côté API).
      const response = await blogApi.getPosts(1, FETCH_ALL_LIMIT);
      setPosts(response.posts);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur de chargement');
    } finally {
      setLoading(false);
    }
  };

  // Compte d'articles par thème curated. Un thème sans aucun article n'est pas affiché
  // (ex: "Jeux" restera discret tant que le blog game design n'a pas tourné). Recalculé
  // quand les posts ou la locale changent (la locale pilote le libellé).
  const themeCounts: TagCount[] = useMemo(
    () =>
      THEMES.map((t) => ({
        tag: t.key,
        label: locale === 'fr' ? t.fr : t.en,
        count: posts.filter((p) => (p.tags || []).includes(t.key)).length,
      })).filter((t) => t.count > 0),
    [posts, locale]
  );

  // Posts visibles selon le filtre. null → tous.
  const filtered = useMemo(
    () => (selectedTag ? posts.filter((p) => (p.tags || []).includes(selectedTag)) : posts),
    [posts, selectedTag]
  );

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="bg-gray-100 dark:bg-gray-800 rounded-xl animate-pulse">
            <div className="h-48 bg-gray-200 dark:bg-gray-700 rounded-t-xl"></div>
            <div className="p-6 space-y-3">
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4"></div>
              <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full"></div>
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-2/3"></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-6 text-center">
        <p className="text-red-600 dark:text-red-400">{error}</p>
        <button
          onClick={fetchPosts}
          className="mt-4 px-4 py-2 bg-red-100 dark:bg-red-800 text-red-700 dark:text-red-200 rounded-lg hover:bg-red-200 dark:hover:bg-red-700 transition-colors"
        >
          Réessayer
        </button>
      </div>
    );
  }

  if (posts.length === 0) {
    return (
      <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-12 text-center">
        <div className="text-6xl mb-4">📝</div>
        <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
          {locale === 'fr' ? 'Aucun article pour le moment' : 'No articles yet'}
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          {locale === 'fr'
            ? 'Les articles seront générés automatiquement depuis les commits.'
            : 'Articles will be automatically generated from commits.'}
        </p>
      </div>
    );
  }

  const articleWord =
    filtered.length > 1
      ? locale === 'fr'
        ? 'articles'
        : 'articles'
      : locale === 'fr'
        ? 'article'
        : 'article';

  return (
    <div className="flex flex-col lg:flex-row gap-8">
      {/* Sidebar de filtrage (gauche desktop / au-dessus en mobile) */}
      <TagFilter
        tags={themeCounts}
        selected={selectedTag}
        total={posts.length}
        locale={locale}
        onSelect={setSelectedTag}
      />

      {/* Colonne principale : compteur de résultats + grille d'articles filtrée */}
      <div className="flex-1 space-y-6">
        <div className="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
          <span data-testid="blog-count">
            {filtered.length} {articleWord}
          </span>
          {selectedTag && (
            // Pastille du filtre actif, cliquable pour réinitialiser.
            <button
              type="button"
              data-testid="active-filter"
              onClick={() => setSelectedTag(null)}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 hover:bg-blue-200 dark:hover:bg-blue-900 transition-colors"
            >
              {selectedTag}
              <span aria-hidden>✕</span>
            </button>
          )}
        </div>

        <div
          data-testid="blog-grid"
          className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-6"
        >
          {filtered.map((post) => (
            <BlogCard key={post.id} post={post} locale={locale} />
          ))}
        </div>
      </div>
    </div>
  );
}
