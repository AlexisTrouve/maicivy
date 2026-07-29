'use client';

import React from 'react';
import Link from 'next/link';
import { BlogPost } from '@/lib/types';

interface BlogCardProps {
  post: BlogPost;
  locale?: string;
}

export function BlogCard({ post, locale = 'fr' }: BlogCardProps) {
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    // toLocaleString avec timezone auto-détectée par le browser
    return date.toLocaleString(locale === 'fr' ? 'fr-FR' : 'en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <Link href={`/${locale}/blog/${post.slug}`} className="block">
    <article data-testid="blog-card" className="group bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden hover:shadow-xl transition-all duration-300 hover:border-blue-300 dark:hover:border-blue-600 cursor-pointer">
      {/* Cover image — SVG/image si dispo, sinon placeholder gradient */}
      {/* Fond fixe #0f172a — SVG ont le même fond, invisibles sur gradient clair en light mode */}
      <div className="h-48 bg-[#0f172a] flex items-center justify-center overflow-hidden">
        {post.cover_image_url ? (
          <img
            src={post.cover_image_url}
            alt={post.title}
            className="w-full h-full object-contain"
          />
        ) : (
          <div className="text-4xl font-bold text-blue-500/30 dark:text-blue-400/30">
            {post.project_name.slice(0, 2).toUpperCase()}
          </div>
        )}
      </div>

      <div className="p-6">
        {/* Tags */}
        <div className="flex flex-wrap gap-2 mb-3">
          <span className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded-full">
            {post.project_name}
          </span>
          {post.tags.slice(0, 2).map((tag) => (
            <span
              key={tag}
              className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full"
            >
              {tag}
            </span>
          ))}
        </div>

        {/* Title */}
        <h2 className="text-xl font-bold text-gray-900 dark:text-white mb-2 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
          {post.title}
        </h2>

        {/* Summary */}
        <p className="text-gray-600 dark:text-gray-400 text-sm mb-4 line-clamp-2">
          {post.summary}
        </p>

        {/* Meta */}
        <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <span>{formatDate(post.published_at || post.created_at)}</span>
          <span className="flex items-center gap-1">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {post.reading_time_minutes} min
          </span>
        </div>
      </div>
    </article>
    </Link>
  );
}
