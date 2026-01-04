'use client';

import React, { useRef, useState } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text, Html } from '@react-three/drei';
import * as THREE from 'three';
import type { Portfolio3DProject, PerformanceLevel } from '@/lib/types';
import type { CardPosition3D } from '@/lib/3d-utils';

interface GlassCard3DProps {
  project: Portfolio3DProject;
  cardPosition: CardPosition3D;
  isSelected: boolean;
  isHovered: boolean;
  onClick: () => void;
  onHover: (hovered: boolean) => void;
  performanceLevel: PerformanceLevel;
}

export function GlassCard3D({
  project,
  cardPosition,
  isSelected,
  isHovered,
  onClick,
  onHover,
  performanceLevel
}: GlassCard3DProps) {
  const groupRef = useRef<THREE.Group>(null);
  const [localHover, setLocalHover] = useState(false);
  const timeOffset = useRef(Math.random() * Math.PI * 2);

  // Calculate scale
  const currentScale = isSelected ? 1.3 : isHovered || localHover ? 1.1 : cardPosition.scale;

  // Floating animation
  useFrame((state) => {
    if (groupRef.current && !isSelected) {
      const t = state.clock.getElapsedTime() + timeOffset.current;
      groupRef.current.position.y = cardPosition.position[1] + Math.sin(t * 0.8) * 0.1;
      groupRef.current.rotation.z = Math.sin(t * 0.5) * 0.02;
    }
  });

  // Card dimensions
  const cardWidth = 1.8;
  const cardHeight = 2.4;
  const cardDepth = 0.08;

  // Category color mapping
  const categoryColors: Record<string, string> = {
    fullstack: '#8b5cf6',
    frontend: '#3b82f6',
    backend: '#10b981',
    mobile: '#f59e0b',
    devops: '#ef4444',
    ai: '#ec4899',
    tools: '#06b6d4',
    demo: '#64748b',
    default: '#6366f1'
  };

  const accentColor = categoryColors[project.category] || categoryColors.default;

  return (
    <group
      ref={groupRef}
      position={cardPosition.position}
      rotation={cardPosition.rotation}
      scale={currentScale}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      onPointerOver={(e) => {
        e.stopPropagation();
        setLocalHover(true);
        onHover(true);
        document.body.style.cursor = 'pointer';
      }}
      onPointerOut={() => {
        setLocalHover(false);
        onHover(false);
        document.body.style.cursor = 'auto';
      }}
    >
      {/* Simple box for debugging - main card */}
      <mesh>
        <boxGeometry args={[cardWidth, cardHeight, cardDepth]} />
        <meshStandardMaterial
          color={accentColor}
          transparent
          opacity={0.9}
        />
      </mesh>

      {/* Content container */}
      <group position={[0, 0, cardDepth / 2 + 0.01]}>
        {/* Category badge */}
        <mesh position={[0, cardHeight / 2 - 0.5, 0]}>
          <planeGeometry args={[1.2, 0.3]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.9} />
        </mesh>

        {/* Featured badge */}
        {project.featured && (
          <mesh position={[cardWidth / 2 - 0.2, cardHeight / 2 - 0.2, 0]}>
            <circleGeometry args={[0.12, 16]} />
            <meshBasicMaterial color="#fbbf24" />
          </mesh>
        )}
      </group>

      {/* HTML overlay for title and details */}
      <Html
        position={[0, 0, cardDepth / 2 + 0.02]}
        center
        distanceFactor={6}
        style={{
          pointerEvents: 'none',
          userSelect: 'none'
        }}
      >
        <div className="text-center w-[160px]">
          <h3 className="text-sm font-bold text-white drop-shadow-lg">
            {project.title}
          </h3>
          <p className="text-xs text-white/80 mt-1">
            {project.category}
          </p>
        </div>
      </Html>
    </group>
  );
}

export default GlassCard3D;
