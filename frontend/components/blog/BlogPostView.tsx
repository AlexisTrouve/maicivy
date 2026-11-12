'use client';

import React, { useState, useEffect, useCallback } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { BlogPost } from '@/lib/types';
import { FollowBlock } from './FollowBlock';
import styles from './article.module.css';

interface BlogPostViewProps {
  post: BlogPost;
  locale?: string;
}

/**
 * BlogPostView — la page de lecture d'un article.
 *
 * QUOI : en-tête (retour, projet, titre serif, chapeau, date et durée), couverture agrandissable,
 * corps (HTML rendu par goldmark côté Go, mis en page par article.module.css), abonnement, puis les
 * commits dont l'article est tiré.
 * POURQUOI cette colonne unique : la mesure de lecture (38rem, ~68 caractères) gouverne tout — en-tête,
 * corps et pied partagent la même largeur, l'œil ne change jamais de marge.
 * COMMENT : composant client pour la seule lightbox ; le HTML de l'article est déjà dans le rendu
 * serveur (props), rien n'est chargé au montage.
 */
export function BlogPostView({ post, locale = 'fr' }: BlogPostViewProps) {
  const t = useTranslations('blog.post');
  const [lightboxOpen, setLightboxOpen] = useState(false);

  // Fermer avec Escape
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') setLightboxOpen(false);
  }, []);

  useEffect(() => {
    if (lightboxOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    } else {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    }
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [lightboxOpen, handleKeyDown]);

  const publishedAt = post.published_at || post.created_at;
  // Locale de la page telle quelle (zh, de, it…) : l'ancienne bascule fr/en affichait l'anglais en zh.
  const date = new Date(publishedAt).toLocaleDateString(locale, { year: 'numeric', month: 'long', day: 'numeric' });

  return (
    <article data-testid="blog-article" className="mx-auto w-full max-w-[38rem]">
      <header>
        <Link
          href={`/${locale}/blog`}
          data-testid="blog-back"
          className="-ml-2 inline-flex min-h-11 items-center gap-2 rounded-md px-2 text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <span aria-hidden="true">←</span>
          {t('back')}
        </Link>

        {/* Projet d'origine : un repère, en petites capitales — le titre porte le sujet. */}
        <p className="mt-8 text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
          {post.project_name}
        </p>

        <h1 className="mt-3 text-[2.15rem] font-semibold leading-[1.08] text-foreground [text-wrap:balance] md:text-5xl">
          {post.title}
        </h1>

        {post.summary && (
          <p className="mt-5 text-lg leading-relaxed text-foreground/75 [text-wrap:pretty] md:text-xl">{post.summary}</p>
        )}

        {/* Méta : date et durée sur une ligne, séparées par un point médian ; filet dessous. */}
        <p className="mt-6 flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-border pb-6 text-sm text-muted-foreground">
          <time dateTime={publishedAt}>{date}</time>
          <span aria-hidden="true">·</span>
          <span>{t('readTime', { minutes: post.reading_time_minutes })}</span>
        </p>
      </header>

      {/* Couverture — un vrai bouton (clavier, lecteur d'écran) qui ouvre la pleine définition. */}
      {post.cover_image_url && (
        <>
          <button
            type="button"
            onClick={() => setLightboxOpen(true)}
            aria-label={t('openImage')}
            data-testid="blog-cover"
            className="relative mt-8 block w-full cursor-zoom-in overflow-hidden rounded-lg border border-border bg-card focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {/* Variante optimisée (AVIF/WebP, à la largeur de l'article) dans l'article ; la pleine
                définition n'est chargée que par la lightbox, au clic. */}
            <span className="relative block aspect-[16/9] w-full">
              <Image
                src={post.cover_image_url}
                alt={post.title}
                fill
                priority
                sizes="(min-width: 768px) 720px, 100vw"
                unoptimized={post.cover_image_url.toLowerCase().endsWith('.svg')}
                className="object-contain"
              />
            </span>
          </button>

          {lightboxOpen && (
            <div
              role="dialog"
              aria-modal="true"
              aria-label={post.title}
              className="fixed inset-0 z-50 flex items-center justify-center bg-background/95 backdrop-blur-sm"
              onClick={() => setLightboxOpen(false)}
            >
              {/* eslint-disable-next-line @next/next/no-img-element -- pleine définition voulue, taille inconnue */}
              <img
                src={post.cover_image_url}
                alt={post.title}
                className="max-h-[90vh] max-w-[92vw] object-contain"
                onClick={(e) => e.stopPropagation()}
              />
              <button
                type="button"
                autoFocus
                className="absolute right-4 top-4 inline-flex h-11 w-11 items-center justify-center rounded-md border border-border bg-card text-foreground/80 transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => setLightboxOpen(false)}
                aria-label={t('closeImage')}
              >
                <span aria-hidden="true" className="text-xl leading-none">✕</span>
              </button>
            </div>
          )}
        </>
      )}

      {/* Corps — la feuille de lecture (article.module.css) s'applique directement au HTML rendu. */}
      {post.content_html ? (
        <div className={`${styles.article} mt-10`} dangerouslySetInnerHTML={{ __html: post.content_html }} />
      ) : (
        <div className={`${styles.article} ${styles.plain} mt-10`}>{post.content}</div>
      )}

      {/* Sujets : en fin de lecture, là où l'on se demande « de quoi d'autre parle-t-il ? ». */}
      {post.tags.length > 0 && (
        <ul aria-label={t('tags')} className="mt-12 flex flex-wrap gap-2">
          {post.tags.map((tag) => (
            <li key={tag} className="rounded-md border border-border px-2.5 py-1 text-xs text-muted-foreground">
              {tag}
            </li>
          ))}
        </ul>
      )}

      {/* Suivre — placé juste après l'article : le lecteur qui vient de finir est exactement celui
          qui veut un moyen de revenir (retour user réel). */}
      <div className="mt-12">
        <FollowBlock />
      </div>

      {/* Commits source : la preuve que l'article raconte du code réel. Grammaire des lignes de
          preuve du dossier vivant — empreinte en monospace, filets entre les lignes. */}
      {post.generated_from_commits && post.generated_from_commits.length > 0 && (
        <footer data-testid="blog-commits" className="mt-16 border-t border-border pt-8">
          <h2 className="text-2xl font-semibold text-foreground">{t('generatedFrom')}</h2>
          <ul className="mt-4">
            {post.generated_from_commits.map((commit) => (
              <li key={commit.sha} className="flex items-baseline gap-4 border-b border-border py-3 text-sm">
                <code className="shrink-0 font-mono text-xs text-muted-foreground">{commit.sha.slice(0, 7)}</code>
                <span className="min-w-0 flex-1 text-foreground/85 [overflow-wrap:anywhere]">{commit.message}</span>
              </li>
            ))}
          </ul>
        </footer>
      )}
    </article>
  );
}
