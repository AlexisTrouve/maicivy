'use client';

import React, { useState, useMemo, useRef, Suspense, useEffect, useCallback } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { OrbitControls, PerspectiveCamera } from '@react-three/drei';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';
import * as THREE from 'three';

import { GlassCard3D } from './GlassCard3D';
import { GlassEnvironment } from './GlassEnvironment';
import { AmbientParticles } from './AmbientParticles';
import { LightRays } from './LightRays';
import { PortfolioNavigation } from './PortfolioNavigation';

import { use3DSupport, use3DQualitySettings } from '@/hooks/use3DSupport';
import type { Portfolio3DProject, PortfolioShowcaseConfig, PerformanceLevel } from '@/lib/types';
import type { CardPosition3D } from '@/lib/3d-utils';

interface PortfolioShowcaseProps {
  projects: Portfolio3DProject[];
  config?: Partial<PortfolioShowcaseConfig>;
  className?: string;
  height?: string;
}

// Camera controller component - zooms in when card is selected
function CameraController({
  selectedIndex,
  cardPositions,
  carouselRotation,
  enabled,
  isSpiralLayout
}: {
  selectedIndex: number | null;
  cardPositions: { position: [number, number, number] }[];
  carouselRotation: number;
  enabled: boolean;
  isSpiralLayout: boolean;
}) {
  const { camera } = useThree();
  const targetPos = useRef(new THREE.Vector3(0, 4, 12));
  const targetLookAt = useRef(new THREE.Vector3(0, 0, 0));

  useEffect(() => {
    if (selectedIndex !== null && cardPositions[selectedIndex]) {
      const cardPos = cardPositions[selectedIndex].position;

      if (isSpiralLayout) {
        // For spiral layout: move camera to face the selected card directly
        const cardVec = new THREE.Vector3(cardPos[0], cardPos[1], cardPos[2]);
        const dir = cardVec.clone().normalize();
        // Position camera outside the card, facing it
        targetPos.current.set(
          cardPos[0] + dir.x * 4,
          cardPos[1] + 1,
          cardPos[2] + dir.z * 4
        );
        targetLookAt.current.set(cardPos[0], cardPos[1], cardPos[2]);
      } else {
        // For circle layout: card is rotated to front
        const radius = 4;
        targetPos.current.set(0, 0.3, radius + 4);
        targetLookAt.current.set(0, 0, radius - 0.5);
      }
    } else {
      // Default overview - see all cards from above
      const overviewHeight = isSpiralLayout ? 6 : 3;
      const overviewDist = isSpiralLayout ? 14 : 10;
      targetPos.current.set(0, overviewHeight, overviewDist);
      targetLookAt.current.set(0, 0, 0);
    }
  }, [selectedIndex, carouselRotation, cardPositions, isSpiralLayout]);

  useFrame(() => {
    if (!enabled) return;

    // Smooth camera transitions
    camera.position.lerp(targetPos.current, 0.05);
    camera.lookAt(targetLookAt.current);
  });

  return null;
}

// Animated carousel group - rotates all cards together with shortest path
function CarouselGroup({
  children,
  rotationY,
  itemCount
}: {
  children: React.ReactNode;
  rotationY: number;
  itemCount: number;
}) {
  const groupRef = useRef<THREE.Group>(null);
  const currentRotation = useRef(0);
  const targetRotation = useRef(0);
  const lastRotationY = useRef(rotationY);

  // Animate rotation with useFrame for smooth interpolation
  useFrame(() => {
    if (!groupRef.current) return;

    // Check if target changed - calculate shortest path
    if (lastRotationY.current !== rotationY) {
      const fullCircle = Math.PI * 2;
      let diff = rotationY - currentRotation.current;

      // Wrap difference to [-π, π] for shortest path
      while (diff > Math.PI) diff -= fullCircle;
      while (diff < -Math.PI) diff += fullCircle;

      targetRotation.current = currentRotation.current + diff;
      lastRotationY.current = rotationY;
    }

    // Lerp towards target rotation
    const delta = targetRotation.current - currentRotation.current;
    if (Math.abs(delta) > 0.001) {
      currentRotation.current += delta * 0.08;
      groupRef.current.rotation.y = currentRotation.current;
    }
  });

  return (
    <group ref={groupRef}>
      {children}
    </group>
  );
}

// Scene content
function PortfolioScene({
  projects,
  config,
  selectedIndex,
  setSelectedIndex,
  performanceLevel
}: {
  projects: Portfolio3DProject[];
  config: PortfolioShowcaseConfig;
  selectedIndex: number | null;
  setSelectedIndex: (index: number | null) => void;
  performanceLevel: PerformanceLevel;
}) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const qualitySettings = use3DQualitySettings();

  // Determine layout type based on project count
  const isSpiralLayout = projects.length > 12;

  // Calculate carousel rotation based on selected index
  // Each card is spaced by (2π / numProjects) radians
  const anglePerCard = (Math.PI * 2) / (isSpiralLayout ? 8 : projects.length);

  // Rotate so selected card faces the camera (at front)
  // Only works for circle layout - spiral layout uses camera movement instead
  const carouselRotation = !isSpiralLayout && selectedIndex !== null
    ? -selectedIndex * anglePerCard
    : 0;

  // Generate card positions - adapts layout based on project count
  const cardPositions = useMemo(() => {
    const count = projects.length;
    const baseRadius = config.radius || 4;

    // For many projects, use spiral layout with multiple rings
    if (count > 12) {
      // Spiral layout for many projects
      const positions: Array<{
        position: [number, number, number];
        rotation: [number, number, number];
        scale: number;
      }> = [];

      const itemsPerRing = 8;
      const ringSpacing = 3;
      const heightSpacing = 0.8;

      for (let i = 0; i < count; i++) {
        const ring = Math.floor(i / itemsPerRing);
        const indexInRing = i % itemsPerRing;
        const angleOffset = ring * (Math.PI / itemsPerRing); // Offset each ring
        const angle = (indexInRing / itemsPerRing) * Math.PI * 2 + angleOffset;
        const radius = baseRadius + ring * ringSpacing;
        const y = -ring * heightSpacing; // Lower rings go down slightly

        positions.push({
          position: [
            Math.sin(angle) * radius,
            y,
            Math.cos(angle) * radius
          ],
          rotation: [0, angle + Math.PI, 0],
          scale: Math.max(0.7, 1 - ring * 0.1) // Slightly smaller in outer rings
        });
      }

      return positions;
    }

    // Simple circle for <= 12 projects
    return projects.map((_, index) => {
      const angle = index * anglePerCard;
      return {
        position: [
          Math.sin(angle) * baseRadius,
          0,
          Math.cos(angle) * baseRadius
        ] as [number, number, number],
        rotation: [0, angle + Math.PI, 0] as [number, number, number],
        scale: 1
      };
    });
  }, [projects.length, config.radius, anglePerCard]);

  return (
    <>
      {/* Camera */}
      <PerspectiveCamera makeDefault position={[0, 1, 8]} fov={50} />

      {/* Camera zooms in when card selected */}
      <CameraController
        selectedIndex={selectedIndex}
        cardPositions={cardPositions}
        carouselRotation={carouselRotation}
        enabled={true}
        isSpiralLayout={isSpiralLayout}
      />

      {/* Orbit controls - only for manual rotation when no selection */}
      <OrbitControls
        enabled={selectedIndex === null}
        enablePan={false}
        enableZoom={true}
        minDistance={5}
        maxDistance={12}
        minPolarAngle={Math.PI / 3}
        maxPolarAngle={Math.PI / 2}
        autoRotate={false}
        dampingFactor={0.05}
        enableDamping
      />

      {/* Environment and lighting */}
      <GlassEnvironment performanceLevel={performanceLevel} />

      {/* Background particles */}
      {config.enableParticles && (
        <AmbientParticles
          count={qualitySettings.particleCount}
          color="#a5b4fc"
          size={0.02}
        />
      )}

      {/* Light rays */}
      {config.enableLightRays && performanceLevel === 'high' && (
        <LightRays color="#8b5cf6" intensity={0.3} />
      )}

      {/* Animated carousel with all cards */}
      <CarouselGroup rotationY={carouselRotation} itemCount={projects.length}>
        {projects.map((project, index) => (
          <GlassCard3D
            key={project.id}
            project={project}
            cardPosition={cardPositions[index]}
            isSelected={selectedIndex === index}
            isHovered={hoveredIndex === index}
            onClick={() => setSelectedIndex(selectedIndex === index ? null : index)}
            onHover={(hovered) => setHoveredIndex(hovered ? index : null)}
            performanceLevel={performanceLevel}
          />
        ))}
      </CarouselGroup>
    </>
  );
}

// 2D Fallback for unsupported devices
function PortfolioFallback2D({ projects }: { projects: Portfolio3DProject[] }) {
  return (
    <div className="w-full h-full bg-gradient-to-br from-slate-900 to-purple-900 p-8 overflow-auto">
      <h2 className="text-2xl font-bold text-white mb-6 text-center">Portfolio</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 max-w-6xl mx-auto">
        {projects.map((project) => (
          <div
            key={project.id}
            className="bg-white/10 backdrop-blur-lg rounded-xl border border-white/20 p-6 text-white"
          >
            <h3 className="text-lg font-semibold mb-2">{project.title}</h3>
            <p className="text-sm text-gray-300 mb-4">{project.description}</p>
            <div className="flex flex-wrap gap-2">
              {project.technologies.slice(0, 3).map((tech) => (
                <span
                  key={tech}
                  className="px-2 py-1 text-xs bg-white/10 rounded"
                >
                  {tech}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// Loading state
function LoadingState() {
  return (
    <div className="w-full h-full flex items-center justify-center bg-slate-900">
      <div className="text-center">
        <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
        <p className="text-white/60">Loading 3D Portfolio...</p>
      </div>
    </div>
  );
}

// Main component
export function PortfolioShowcase({
  projects,
  config: userConfig,
  className = '',
  height = '100vh'
}: PortfolioShowcaseProps) {
  const { isSupported, performanceLevel } = use3DSupport();
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  // Merge config with defaults based on performance
  const config: PortfolioShowcaseConfig = useMemo(() => ({
    layout: userConfig?.layout || 'circular',
    radius: userConfig?.radius || 4,
    cardSpacing: userConfig?.cardSpacing || 2.5,
    enablePostProcessing: userConfig?.enablePostProcessing ?? performanceLevel === 'high',
    enableParticles: userConfig?.enableParticles ?? performanceLevel !== 'low',
    enableLightRays: userConfig?.enableLightRays ?? performanceLevel === 'high'
  }), [userConfig, performanceLevel]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (selectedIndex === null) return;

    if (e.key === 'ArrowLeft') {
      setSelectedIndex(selectedIndex > 0 ? selectedIndex - 1 : projects.length - 1);
    } else if (e.key === 'ArrowRight') {
      setSelectedIndex(selectedIndex < projects.length - 1 ? selectedIndex + 1 : 0);
    } else if (e.key === 'Escape') {
      setSelectedIndex(null);
    }
  }, [selectedIndex, projects.length]);

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  // Loading state
  if (!mounted) {
    return <LoadingState />;
  }

  // Fallback for unsupported devices
  if (!isSupported) {
    return <PortfolioFallback2D projects={projects} />;
  }

  return (
    <div
      className={`relative ${className}`}
      style={{ height, background: 'linear-gradient(to bottom right, #0f0f23, #1e1b4b)' }}
    >
      <Canvas
        gl={{
          antialias: performanceLevel !== 'low',
          alpha: true,
          powerPreference: performanceLevel === 'high' ? 'high-performance' : 'default',
          preserveDrawingBuffer: true
        }}
        dpr={performanceLevel === 'high' ? [1, 2] : [1, 1]}
        onCreated={({ gl }) => {
          console.log('Canvas created, WebGL context ready');
          gl.setClearColor(0x0f0f23, 0);
        }}
        fallback={<LoadingState />}
      >
        <PortfolioScene
          projects={projects}
          config={config}
          selectedIndex={selectedIndex}
          setSelectedIndex={setSelectedIndex}
          performanceLevel={performanceLevel}
        />
      </Canvas>

      {/* HTML Navigation overlay */}
      <PortfolioNavigation
        projects={projects}
        currentIndex={selectedIndex}
        onNavigate={setSelectedIndex}
        onClose={() => setSelectedIndex(null)}
      />
    </div>
  );
}

export default PortfolioShowcase;
