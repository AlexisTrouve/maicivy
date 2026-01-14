/**
 * Wrapper React Three Fiber
 * Configuration Canvas, Camera, Lights, Controls
 * Responsive, Performance monitoring, Fallback WebGL
 */

'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, PerspectiveCamera } from '@react-three/drei';
import { use3DSupport, use3DQualitySettings } from '@/hooks/use3DSupport';
import { FPSMonitor } from '@/lib/3d-utils';
import type { Scene3DConfig } from '@/lib/types';

interface Scene3DWrapperProps {
  children: React.ReactNode;
  config?: Scene3DConfig;
  showFPS?: boolean;
  fallback?: React.ReactNode;
  className?: string;
  cameraPosition?: [number, number, number];
  enableControls?: boolean;
}

/**
 * Wrapper principal pour scènes 3D
 */
export function Scene3DWrapper({
  children,
  config = {},
  showFPS = false,
  fallback,
  className = '',
  cameraPosition = [0, 0, 5],
  enableControls = true,
}: Scene3DWrapperProps) {
  const { isSupported, performanceLevel, reason } = use3DSupport();
  const qualitySettings = use3DQualitySettings();
  const [fps, setFps] = useState(60);
  const [fpsMonitor] = useState(() => new FPSMonitor());

  // Monitor FPS si demandé
  useEffect(() => {
    if (!showFPS || !isSupported) return;

    const interval = setInterval(() => {
      const currentFps = fpsMonitor.getFPS();
      setFps(currentFps);
    }, 1000);

    return () => clearInterval(interval);
  }, [showFPS, isSupported, fpsMonitor]);

  // Fallback si WebGL non supporté
  if (!isSupported) {
    if (fallback) {
      return <>{fallback}</>;
    }

    return (
      <div className={`flex items-center justify-center bg-gray-100 rounded-lg ${className}`}>
        <div className="text-center p-8">
          <div className="text-4xl mb-4">🎨</div>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">
            Effets 3D non disponibles
          </h3>
          <p className="text-sm text-gray-600">
            {reason || 'Votre navigateur ne supporte pas WebGL'}
          </p>
        </div>
      </div>
    );
  }

  // Configuration Canvas selon performance
  const canvasConfig = {
    antialias: config.antialias ?? qualitySettings.antialias,
    alpha: config.alpha ?? true,
    powerPreference: config.powerPreference ?? 'high-performance',
    dpr: config.pixelRatio ?? qualitySettings.pixelRatio,
    shadows: config.shadows ?? qualitySettings.shadows,
  };

  return (
    <div className={`relative ${className}`}>
      <Canvas
        {...canvasConfig}
        gl={{
          antialias: canvasConfig.antialias,
          alpha: canvasConfig.alpha,
          powerPreference: canvasConfig.powerPreference as WebGLPowerPreference,
        }}
        onCreated={({ gl }) => {
          // Optimisations rendering
          gl.setClearColor(0x000000, 0);
        }}
      >
        <PerspectiveCamera makeDefault position={cameraPosition} fov={75} />

        {/* Lumières */}
        <ambientLight intensity={0.5} />
        <pointLight position={[10, 10, 10]} intensity={1} />
        <pointLight position={[-10, -10, 10]} intensity={0.5} color="#a78bfa" />

        {/* Controls */}
        {enableControls && (
          <OrbitControls
            enableDamping
            dampingFactor={0.05}
            enableZoom={true}
            enablePan={false}
            minDistance={2}
            maxDistance={10}
            minPolarAngle={Math.PI / 4}
            maxPolarAngle={Math.PI / 1.5}
          />
        )}

        {/* Contenu 3D */}
        <Suspense
          fallback={
            <mesh>
              <boxGeometry args={[1, 1, 1]} />
              <meshBasicMaterial color="#3b82f6" wireframe />
            </mesh>
          }
        >
          {children}
        </Suspense>
      </Canvas>

      {/* FPS Counter */}
      {showFPS && (
        <div className="absolute top-4 right-4 bg-black/70 text-white text-xs px-3 py-1 rounded-full font-mono">
          {fps} FPS
          <span className="ml-2 text-gray-400">
            ({performanceLevel})
          </span>
        </div>
      )}

      {/* Performance warning */}
      {performanceLevel === 'low' && (
        <div className="absolute bottom-4 left-4 bg-yellow-500/90 text-white text-xs px-3 py-2 rounded max-w-xs">
          ⚠️ Performances limitées détectées. Effets 3D simplifiés.
        </div>
      )}
    </div>
  );
}

/**
 * Wrapper simplifié sans controls
 */
export function SimpleScene3D({
  children,
  className = '',
  height = '400px',
}: {
  children: React.ReactNode;
  className?: string;
  height?: string;
}) {
  return (
    <div className={className} style={{ height }}>
      <Scene3DWrapper enableControls={false}>
        {children}
      </Scene3DWrapper>
    </div>
  );
}
