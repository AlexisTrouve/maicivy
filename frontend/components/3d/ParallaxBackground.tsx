/**
 * Background 3D avec particules et parallax
 * Stars, geometric shapes flottantes
 * Parallax au scroll, performance optimisée
 */

'use client';

import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { Points, PointMaterial } from '@react-three/drei';
import * as THREE from 'three';
import { Scene3DWrapper } from './Scene3DWrapper';
import { generateParticlePositions, generateSpiralPositions } from '@/lib/3d-utils';
import { use3DQualitySettings } from '@/hooks/use3DSupport';

interface ParticlesSystemProps {
  count: number;
  pattern?: 'random' | 'spiral';
  color?: string;
  size?: number;
  speed?: number;
}

/**
 * Système de particules (étoiles)
 */
function ParticlesSystem({
  count,
  pattern = 'random',
  color = '#ffffff',
  size = 0.02,
  speed = 0.5,
}: ParticlesSystemProps) {
  const pointsRef = useRef<THREE.Points>(null);

  // Générer positions selon pattern
  const positions = useMemo(() => {
    if (pattern === 'spiral') {
      return generateSpiralPositions(count, 5, 10);
    }
    return generateParticlePositions(count, { x: 20, y: 20, z: 20 });
  }, [count, pattern]);

  // Animation rotation
  useFrame((state) => {
    if (!pointsRef.current) return;

    const time = state.clock.getElapsedTime();

    // Rotation lente
    pointsRef.current.rotation.y = time * 0.05 * speed;
    pointsRef.current.rotation.x = Math.sin(time * 0.1) * 0.1;
  });

  return (
    <Points ref={pointsRef} positions={positions} stride={3} frustumCulled>
      <PointMaterial
        transparent
        color={color}
        size={size}
        sizeAttenuation={true}
        depthWrite={false}
        opacity={0.8}
      />
    </Points>
  );
}

interface FloatingShapesProps {
  count?: number;
}

/**
 * Formes géométriques flottantes
 */
function FloatingShapes({ count = 10 }: FloatingShapesProps) {
  const groupRef = useRef<THREE.Group>(null);

  // Générer positions et configs
  const shapes = useMemo(() => {
    return Array.from({ length: count }, (_, i) => ({
      id: i,
      type: ['box', 'sphere', 'torus'][Math.floor(Math.random() * 3)],
      position: [
        (Math.random() - 0.5) * 15,
        (Math.random() - 0.5) * 15,
        (Math.random() - 0.5) * 15,
      ] as [number, number, number],
      color: ['#3b82f6', '#8b5cf6', '#ec4899', '#10b981'][Math.floor(Math.random() * 4)],
      scale: 0.3 + Math.random() * 0.5,
      speed: 0.5 + Math.random() * 1,
    }));
  }, [count]);

  useFrame((state) => {
    if (!groupRef.current) return;

    const time = state.clock.getElapsedTime();

    // Animer chaque shape
    groupRef.current.children.forEach((child, i) => {
      const shape = shapes[i];
      if (!shape) return;

      // Float motion
      child.position.y += Math.sin(time * shape.speed + i) * 0.002;
      child.rotation.x += 0.003 * shape.speed;
      child.rotation.y += 0.005 * shape.speed;
    });
  });

  return (
    <group ref={groupRef}>
      {shapes.map((shape) => {
        let geometry;
        switch (shape.type) {
          case 'box':
            geometry = <boxGeometry args={[1, 1, 1]} />;
            break;
          case 'sphere':
            geometry = <sphereGeometry args={[0.5, 16, 16]} />;
            break;
          case 'torus':
            geometry = <torusGeometry args={[0.5, 0.2, 16, 32]} />;
            break;
          default:
            geometry = <boxGeometry args={[1, 1, 1]} />;
        }

        return (
          <mesh key={shape.id} position={shape.position} scale={shape.scale}>
            {geometry}
            <meshStandardMaterial
              color={shape.color}
              metalness={0.8}
              roughness={0.2}
              transparent
              opacity={0.3}
              wireframe
            />
          </mesh>
        );
      })}
    </group>
  );
}

interface ParallaxBackgroundSceneProps {
  particleCount: number;
  showShapes?: boolean;
  variant?: 'stars' | 'spiral' | 'mixed';
}

/**
 * Scène complète background
 */
function ParallaxBackgroundScene({
  particleCount,
  showShapes = true,
  variant = 'stars',
}: ParallaxBackgroundSceneProps) {
  return (
    <>
      {/* Particules principales */}
      {variant === 'stars' && (
        <ParticlesSystem count={particleCount} pattern="random" size={0.02} speed={0.5} />
      )}

      {variant === 'spiral' && (
        <ParticlesSystem
          count={particleCount}
          pattern="spiral"
          color="#8b5cf6"
          size={0.03}
          speed={1}
        />
      )}

      {variant === 'mixed' && (
        <>
          <ParticlesSystem count={particleCount / 2} pattern="random" size={0.02} speed={0.3} />
          <ParticlesSystem
            count={particleCount / 2}
            pattern="spiral"
            color="#a78bfa"
            size={0.025}
            speed={0.7}
          />
        </>
      )}

      {/* Formes flottantes */}
      {showShapes && <FloatingShapes count={8} />}
    </>
  );
}

interface ParallaxBackgroundProps {
  className?: string;
  height?: string;
  variant?: 'stars' | 'spiral' | 'mixed';
  showShapes?: boolean;
  showFPS?: boolean;
}

/**
 * Composant principal ParallaxBackground
 */
export function ParallaxBackground({
  className = '',
  height = '100vh',
  variant = 'stars',
  showShapes = true,
  showFPS = false,
}: ParallaxBackgroundProps) {
  const qualitySettings = use3DQualitySettings();

  return (
    <div
      className={`fixed inset-0 -z-10 ${className}`}
      style={{ height, width: '100%', pointerEvents: 'none' }}
    >
      <Scene3DWrapper
        showFPS={showFPS}
        cameraPosition={[0, 0, 5]}
        enableControls={false}
        config={{
          alpha: true,
          antialias: qualitySettings.antialias,
        }}
      >
        <ParallaxBackgroundScene
          particleCount={qualitySettings.particleCount}
          showShapes={showShapes}
          variant={variant}
        />
      </Scene3DWrapper>
    </div>
  );
}

/**
 * Variante overlay (au-dessus du contenu)
 */
export function ParallaxOverlay({
  className = '',
  opacity = 0.5,
  variant = 'stars',
}: {
  className?: string;
  opacity?: number;
  variant?: 'stars' | 'spiral' | 'mixed';
}) {
  return (
    <div
      className={`absolute inset-0 pointer-events-none ${className}`}
      style={{ opacity }}
    >
      <ParallaxBackground variant={variant} showShapes={false} />
    </div>
  );
}

/**
 * Background minimaliste (performances optimales)
 */
export function MinimalBackground({ className = '' }) {
  const qualitySettings = use3DQualitySettings();

  return (
    <div className={`fixed inset-0 -z-10 ${className}`} style={{ pointerEvents: 'none' }}>
      <Scene3DWrapper cameraPosition={[0, 0, 5]} enableControls={false}>
        <ParticlesSystem
          count={Math.min(qualitySettings.particleCount, 300)}
          pattern="random"
          size={0.02}
          speed={0.3}
        />
      </Scene3DWrapper>
    </div>
  );
}
