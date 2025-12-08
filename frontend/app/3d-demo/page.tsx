/**
 * Page de démonstration des effets 3D
 * Test des composants Avatar3D, SkillsGraph3D, ParallaxBackground
 */

'use client';

import React, { useState } from 'react';
import {
  Avatar3D,
  AvatarCube3D,
  AvatarMultiShape3D,
  SkillsGraph3DDemo,
  ParallaxBackground,
  MinimalBackground,
} from '@/components/3d';
import { use3DSupport } from '@/hooks/use3DSupport';

export default function Demo3DPage() {
  const { isSupported, performanceLevel, webGLVersion, isMobile, reason } = use3DSupport();
  const [showParallax, setShowParallax] = useState(false);
  const [parallaxVariant, setParallaxVariant] = useState<'stars' | 'spiral' | 'mixed'>('stars');

  return (
    <main className="min-h-screen bg-gray-50">
      {/* Parallax Background (optionnel) */}
      {showParallax && <ParallaxBackground variant={parallaxVariant} showFPS />}

      <div className="container mx-auto px-4 py-12 relative z-10">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">
            Démonstration Effets 3D
          </h1>
          <p className="text-gray-600 mb-6">
            Explorez les visualisations 3D interactives du projet maicivy
          </p>

          {/* Support Info */}
          <div className="inline-flex items-center gap-4 bg-white rounded-lg shadow-sm p-4 mb-8">
            <div className="text-left">
              <div className="text-sm font-semibold text-gray-700 mb-1">
                WebGL Support
              </div>
              <div className="flex items-center gap-2">
                {isSupported ? (
                  <>
                    <span className="text-green-500">✓</span>
                    <span className="text-sm text-gray-600">
                      WebGL {webGLVersion} • Performance: {performanceLevel}
                      {isMobile && ' • Mobile'}
                    </span>
                  </>
                ) : (
                  <>
                    <span className="text-red-500">✗</span>
                    <span className="text-sm text-gray-600">{reason}</span>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Controls Parallax */}
        <div className="bg-white rounded-lg shadow-sm p-6 mb-8">
          <h2 className="text-xl font-semibold mb-4">Parallax Background</h2>
          <div className="flex flex-wrap gap-4 items-center">
            <button
              onClick={() => setShowParallax(!showParallax)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                showParallax
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
              }`}
            >
              {showParallax ? 'Désactiver' : 'Activer'} Parallax
            </button>

            {showParallax && (
              <>
                <div className="flex gap-2">
                  {(['stars', 'spiral', 'mixed'] as const).map((variant) => (
                    <button
                      key={variant}
                      onClick={() => setParallaxVariant(variant)}
                      className={`px-3 py-1 rounded text-sm font-medium transition-colors ${
                        parallaxVariant === variant
                          ? 'bg-purple-500 text-white'
                          : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                      }`}
                    >
                      {variant}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>

        {/* Grid Components */}
        <div className="grid md:grid-cols-2 gap-8">
          {/* Avatar 3D */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-xl font-semibold mb-4">Avatar 3D - Icosahedron</h2>
            <p className="text-sm text-gray-600 mb-4">
              Géométrie low-poly avec rotation interactive. Survolez avec la souris.
            </p>
            <Avatar3D height="400px" showFPS />
          </div>

          {/* Avatar Cube */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-xl font-semibold mb-4">Avatar 3D - Cube</h2>
            <p className="text-sm text-gray-600 mb-4">
              Variante minimaliste avec rotation automatique.
            </p>
            <AvatarCube3D height="400px" />
          </div>

          {/* Avatar Multi-Shape */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-xl font-semibold mb-4">Avatar Multi-Formes</h2>
            <p className="text-sm text-gray-600 mb-4">
              Composition de plusieurs géométries (sphère + anneaux).
            </p>
            <AvatarMultiShape3D height="400px" />
          </div>

          {/* Skills Graph Placeholder */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-xl font-semibold mb-4">Skills Graph 3D</h2>
            <p className="text-sm text-gray-600 mb-4">
              Graph interactif des compétences. Visible dans la section ci-dessous.
            </p>
            <div className="h-[400px] bg-gradient-to-br from-indigo-100 to-purple-100 rounded-lg flex items-center justify-center">
              <div className="text-center">
                <div className="text-6xl mb-4">⬇️</div>
                <p className="text-gray-600">Voir section complète ci-dessous</p>
              </div>
            </div>
          </div>
        </div>

        {/* Skills Graph 3D - Full Width */}
        <div className="mt-8 bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-2xl font-semibold mb-4">
            Skills Graph 3D - Réseau de Compétences
          </h2>
          <p className="text-gray-600 mb-6">
            Visualisation interactive des compétences en 3D. Les sphères représentent les skills
            (taille = niveau), les lignes représentent les relations entre catégories.
          </p>
          <SkillsGraph3DDemo height="600px" />
        </div>

        {/* Info Cards */}
        <div className="mt-12 grid md:grid-cols-3 gap-6">
          <div className="bg-blue-50 rounded-lg p-6">
            <div className="text-3xl mb-3">⚡</div>
            <h3 className="font-semibold text-gray-900 mb-2">Performance Optimisée</h3>
            <p className="text-sm text-gray-600">
              Détection automatique du device. Les effets s'adaptent selon les performances.
            </p>
          </div>

          <div className="bg-purple-50 rounded-lg p-6">
            <div className="text-3xl mb-3">📱</div>
            <h3 className="font-semibold text-gray-900 mb-2">Responsive</h3>
            <p className="text-sm text-gray-600">
              Compatible desktop, tablet, mobile. Dégradation gracieuse sur devices low-end.
            </p>
          </div>

          <div className="bg-green-50 rounded-lg p-6">
            <div className="text-3xl mb-3">🎨</div>
            <h3 className="font-semibold text-gray-900 mb-2">Interactif</h3>
            <p className="text-sm text-gray-600">
              Controls souris/touch, hover effects, click interactions. Expérience immersive.
            </p>
          </div>
        </div>

        {/* Technical Details */}
        <div className="mt-12 bg-gray-900 rounded-lg p-8 text-white">
          <h2 className="text-2xl font-semibold mb-4">Détails Techniques</h2>
          <div className="grid md:grid-cols-2 gap-6">
            <div>
              <h3 className="font-semibold mb-2 text-blue-400">Technologies</h3>
              <ul className="space-y-1 text-sm text-gray-300">
                <li>• Three.js (3D engine)</li>
                <li>• @react-three/fiber (React renderer)</li>
                <li>• @react-three/drei (Helpers)</li>
                <li>• @react-spring/three (Animations)</li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold mb-2 text-purple-400">Optimisations</h3>
              <ul className="space-y-1 text-sm text-gray-300">
                <li>• Instanced rendering pour particules</li>
                <li>• Frustum culling automatique</li>
                <li>• LOD selon performance device</li>
                <li>• Lazy loading composants 3D</li>
              </ul>
            </div>
          </div>

          <div className="mt-6 p-4 bg-gray-800 rounded">
            <div className="text-sm font-mono text-gray-400">
              <div>WebGL Version: {webGLVersion || 'N/A'}</div>
              <div>Performance Level: {performanceLevel}</div>
              <div>Device: {isMobile ? 'Mobile' : 'Desktop'}</div>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
