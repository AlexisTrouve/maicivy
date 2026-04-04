'use client';

import React, { useState, useEffect } from 'react';
import nextDynamic from 'next/dynamic';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { use3DSupport } from '@/hooks/use3DSupport';
import type { Portfolio3DProject } from '@/lib/types';

const MAIPROFILES_URL = 'https://maiprofiles.etheryale.com';

// Lazy load 3D components — SSR désactivé (WebGL client-only)
const PortfolioShowcase = nextDynamic(
  () => import('@/components/3d/portfolio').then(m => m.PortfolioShowcase),
  { ssr: false }
);

// Mapping catégorie maiprofiles → catégorie portfolio 3D
function mapCategory(cat: string): string {
  const lower = cat.toLowerCase();
  if (lower.includes('web') || lower.includes('saas') || lower.includes('full')) return 'fullstack';
  if (lower.includes('backend') || lower.includes('api')) return 'backend';
  if (lower.includes('frontend') || lower.includes('ui')) return 'frontend';
  if (lower.includes('ai') || lower.includes('ml') || lower.includes('nlp')) return 'ai';
  if (lower.includes('devops') || lower.includes('infra') || lower.includes('tool')) return 'devops';
  if (lower.includes('game') || lower.includes('engine')) return 'devops';
  if (lower.includes('mobile') || lower.includes('flutter')) return 'mobile';
  return 'tools';
}

// Transforme un projet maiprofiles en Portfolio3DProject
function toPortfolio3D(p: any): Portfolio3DProject {
  return {
    id: p.id,
    title: p.name,
    description: p.description?.portfolio || p.description?.short || '',
    // Image hero si dispo, sinon placeholder basé sur l'id
    imageUrl: p.screenshots?.[0] || `/projects/${p.id}.png`,
    technologies: p.stack || [],
    githubUrl: p.links?.github,
    demoUrl: p.links?.live,
    featured: p.status === 'production',
    category: mapCategory(p.category || ''),
  };
}

export default function Demo3DContent() {
  const [mounted, setMounted] = useState(false);
  const [projects, setProjects] = useState<Portfolio3DProject[]>([]);
  const [loading, setLoading] = useState(true);
  const [showLegacy, setShowLegacy] = useState(false);
  const { isSupported, performanceLevel, webGLVersion, isMobile } = use3DSupport();

  // Fetch projets depuis maiprofiles
  useEffect(() => {
    setMounted(true);
    fetch(`${MAIPROFILES_URL}/projects`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (Array.isArray(data) && data.length > 0) {
          setProjects(data.map(toPortfolio3D));
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  // Attendre le montage + le fetch
  if (!mounted || loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-white/60">Loading 3D Portfolio...</p>
        </div>
      </div>
    );
  }

  // Aucun projet chargé — erreur réseau ou API vide
  if (projects.length === 0) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <p className="text-white/60 mb-4">Aucun projet disponible.</p>
          <Link href="/" className="text-purple-400 hover:text-purple-300 underline">
            Retour
          </Link>
        </div>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="absolute top-0 left-0 right-0 z-20 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              href="/"
              className="p-2 rounded-lg bg-white/10 backdrop-blur border border-white/20 text-white hover:bg-white/20 transition-colors"
              title="Retour à l'accueil"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
            <div>
              <h1 className="text-2xl font-bold text-white">Portfolio 3D</h1>
              <p className="text-sm text-white/60">
                WebGL {webGLVersion} | {performanceLevel} | {isMobile ? 'Mobile' : 'Desktop'}
                {' '}| {projects.length} projets
              </p>
            </div>
          </div>
          <button
            onClick={() => setShowLegacy(!showLegacy)}
            className="px-4 py-2 rounded-lg bg-white/10 backdrop-blur border border-white/20 text-white text-sm hover:bg-white/20 transition-colors"
          >
            {showLegacy ? 'Show Portfolio' : 'Show Legacy Demo'}
          </button>
        </div>
      </div>

      {/* Main Content — projets dynamiques depuis maiprofiles */}
      {!showLegacy ? (
        <PortfolioShowcase
          projects={projects}
          height="100vh"
          config={{
            layout: projects.length > 12 ? 'spiral' : 'circular',
            radius: 4,
            enablePostProcessing: performanceLevel === 'high',
            enableParticles: performanceLevel !== 'low',
            enableLightRays: performanceLevel === 'high'
          }}
        />
      ) : (
        <LegacyDemo />
      )}

      {/* Support Info (bottom right) */}
      <div className="absolute bottom-4 right-4 z-20">
        <div className="bg-black/40 backdrop-blur-sm rounded-lg px-3 py-2 text-xs text-white/60">
          {isSupported ? (
            <span className="text-green-400">3D Supported</span>
          ) : (
            <span className="text-red-400">3D Not Supported</span>
          )}
        </div>
      </div>
    </main>
  );
}

// Legacy demo for comparison
function LegacyDemo() {
  const Avatar3D = nextDynamic(() => import('@/components/3d').then(m => m.Avatar3D), { ssr: false });
  const SkillsGraph3DDemo = nextDynamic(() => import('@/components/3d').then(m => m.SkillsGraph3DDemo), { ssr: false });

  return (
    <div className="pt-20 pb-12 px-4">
      <div className="max-w-7xl mx-auto">
        <div className="grid md:grid-cols-2 gap-8">
          <div className="bg-white/5 backdrop-blur rounded-xl p-6 border border-white/10">
            <h2 className="text-xl font-semibold text-white mb-4">Avatar 3D</h2>
            <Avatar3D height="400px" showFPS />
          </div>
          <div className="bg-white/5 backdrop-blur rounded-xl p-6 border border-white/10">
            <h2 className="text-xl font-semibold text-white mb-4">Skills Graph 3D</h2>
            <SkillsGraph3DDemo height="400px" />
          </div>
        </div>
      </div>
    </div>
  );
}
