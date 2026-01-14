'use client';

import React, { useState, useEffect } from 'react';
import nextDynamic from 'next/dynamic';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { use3DSupport } from '@/hooks/use3DSupport';
import type { Portfolio3DProject } from '@/lib/types';

// Lazy load 3D components
const PortfolioShowcase = nextDynamic(
  () => import('@/components/3d/portfolio').then(m => m.PortfolioShowcase),
  { ssr: false }
);

// Real projects data
const demoProjects: Portfolio3DProject[] = [
  {
    id: '1',
    title: 'maicivy',
    description: 'CV interactif intelligent avec generation de lettres de motivation par IA. Stack moderne avec Next.js 14, Go, et Three.js pour les effets 3D.',
    imageUrl: '/projects/maicivy.png',
    technologies: ['Next.js', 'Go', 'Three.js', 'PostgreSQL', 'Redis'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: true,
    category: 'fullstack'
  },
  {
    id: '2',
    title: 'GroveEngine',
    description: 'Moteur C++ modulaire avec hot-reload ultra-rapide (0.4ms). Architecture optimisee pour le developpement avec Claude Code et iteration rapide.',
    imageUrl: '/projects/groveengine.png',
    technologies: ['C++', 'CMake', 'ImGui', 'Hot-Reload'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: true,
    category: 'devops'
  },
  {
    id: '3',
    title: 'VBA MCP Server',
    description: 'Serveur MCP pour extraction, analyse et injection de code VBA dans les fichiers Office. 24 outils pour automatiser Excel, Word et Access avec Claude.',
    imageUrl: '/projects/vba-mcp.png',
    technologies: ['TypeScript', 'MCP', 'COM', 'Office'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: true,
    category: 'devops'
  },
  {
    id: '4',
    title: 'Freelance Dashboard',
    description: 'Demo VBA MCP - Dashboard Excel pour suivi freelance avec KPIs, tableaux croises dynamiques et automatisation VBA.',
    imageUrl: '/projects/freelance-dashboard.png',
    technologies: ['Excel', 'VBA', 'MCP Demo'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: false,
    category: 'demo'
  },
  {
    id: '5',
    title: 'TimeTrack Pro',
    description: 'Demo VBA MCP - Gestionnaire de temps Access avec suivi heures par client/projet. Vitrine des capacites Access du serveur MCP.',
    imageUrl: '/projects/timetrack.png',
    technologies: ['Access', 'VBA', 'SQL', 'MCP Demo'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: false,
    category: 'demo'
  },
  {
    id: '6',
    title: 'Confluent',
    description: 'Langue construite complete pour un univers JDR. Systeme linguistique (67 racines, grammaire SOV), API de traduction multi-LLM et interface web temps reel.',
    imageUrl: '/projects/confluent.png',
    technologies: ['Node.js', 'Claude API', 'OpenAI', 'Linguistics'],
    // githubUrl: '', // TODO: Add GitHub URL
    featured: true,
    category: 'ai'
  }
];

export default function Demo3DContent() {
  const [mounted, setMounted] = useState(false);
  const [showLegacy, setShowLegacy] = useState(false);
  const { isSupported, performanceLevel, webGLVersion, isMobile } = use3DSupport();

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-white/60">Loading 3D Portfolio...</p>
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

      {/* Main Content */}
      {!showLegacy ? (
        // New Portfolio Showcase
        <PortfolioShowcase
          projects={demoProjects}
          height="100vh"
          config={{
            layout: 'circular',
            radius: 4,
            enablePostProcessing: performanceLevel === 'high',
            enableParticles: performanceLevel !== 'low',
            enableLightRays: performanceLevel === 'high'
          }}
        />
      ) : (
        // Legacy demo content
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
