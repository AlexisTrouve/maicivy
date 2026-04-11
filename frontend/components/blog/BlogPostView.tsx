'use client';

import React, { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { BlogPost } from '@/lib/types';

interface BlogPostViewProps {
  post: BlogPost;
  locale?: string;
}

export function BlogPostView({ post, locale = 'fr' }: BlogPostViewProps) {
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

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  return (
    <article className="max-w-3xl mx-auto">
      {/* Header */}
      <header className="mb-8">
        {/* Back link */}
        <Link
          href={`/${locale}/blog`}
          className="inline-flex items-center gap-2 text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 mb-6 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          {locale === 'fr' ? 'Retour au blog' : 'Back to blog'}
        </Link>

        {/* Tags */}
        <div className="flex flex-wrap gap-2 mb-4">
          <span className="text-sm px-3 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded-full font-medium">
            {post.project_name}
          </span>
          {post.tags.map((tag) => (
            <span
              key={tag}
              className="text-sm px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full"
            >
              {tag}
            </span>
          ))}
        </div>

        {/* Cover image — cliquable pour ouvrir en lightbox */}
        {post.cover_image_url && (
          <>
            <div
              {/* Fond fixe #0f172a — assure la visibilité des SVG (même fond) en light et dark mode */}
              className="mb-6 rounded-xl overflow-hidden bg-[#0f172a] cursor-zoom-in"
              onClick={() => setLightboxOpen(true)}
            >
              <img
                src={post.cover_image_url}
                alt={post.title}
                className="w-full max-h-80 object-contain"
              />
            </div>

            {/* Lightbox */}
            {lightboxOpen && (
              <div
                className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm"
                onClick={() => setLightboxOpen(false)}
              >
                <img
                  src={post.cover_image_url}
                  alt={post.title}
                  className="max-w-[90vw] max-h-[90vh] object-contain"
                  onClick={(e) => e.stopPropagation()}
                />
                <button
                  className="absolute top-4 right-4 text-white/70 hover:text-white transition-colors"
                  onClick={() => setLightboxOpen(false)}
                  aria-label="Fermer"
                >
                  <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            )}
          </>
        )}

        {/* Title */}
        <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
          {post.title}
        </h1>

        {/* Summary */}
        <p className="text-xl text-gray-600 dark:text-gray-400 mb-6">
          {post.summary}
        </p>

        {/* Meta */}
        <div className="flex items-center gap-6 text-sm text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700 pb-6">
          <span className="flex items-center gap-2">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            {formatDate(post.published_at || post.created_at)}
          </span>
          <span className="flex items-center gap-2">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {post.reading_time_minutes} min {locale === 'fr' ? 'de lecture' : 'read'}
          </span>
        </div>
      </header>

      {/* Content */}
      <div className="prose prose-lg dark:prose-invert max-w-none mb-12">
        {post.content_html ? (
          <div dangerouslySetInnerHTML={{ __html: post.content_html }} />
        ) : (
          <div className="whitespace-pre-wrap">{post.content}</div>
        )}
      </div>

      {/* Commits source */}
      {post.generated_from_commits && post.generated_from_commits.length > 0 && (
        <footer className="border-t border-gray-200 dark:border-gray-700 pt-8">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            {locale === 'fr' ? 'Généré depuis ces commits' : 'Generated from these commits'}
          </h3>
          <div className="space-y-2">
            {post.generated_from_commits.map((commit) => (
              <div
                key={commit.sha}
                className="flex items-start gap-3 text-sm bg-gray-50 dark:bg-gray-800 rounded-lg p-3"
              >
                <code className="text-xs text-gray-500 dark:text-gray-400 font-mono bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                  {commit.sha.slice(0, 7)}
                </code>
                <span className="text-gray-700 dark:text-gray-300 flex-1">
                  {commit.message}
                </span>
              </div>
            ))}
          </div>
        </footer>
      )}
    </article>
  );
}
