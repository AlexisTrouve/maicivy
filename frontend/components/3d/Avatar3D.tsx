/**
 * Avatar 3D simple avec rotation interactive
 * Géométrie low-poly (icosahedron), lighting dynamique
 * Rotation au hover souris
 */

'use client';

import React, { useRef, useState } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import { useSpring, animated } from '@react-spring/three';
import * as THREE from 'three';
import { Scene3DWrapper } from './Scene3DWrapper';

interface Avatar3DMeshProps {
  color?: string;
  metalness?: number;
  roughness?: number;
}

/**
 * Mesh Avatar avec rotation interactive
 */
function Avatar3DMesh({
  color = '#3b82f6',
  metalness = 0.7,
  roughness = 0.3,
}: Avatar3DMeshProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  const { viewport } = useThree();

  // Animation spring au hover
  const { scale } = useSpring({
    scale: hovered ? 1.1 : 1,
    config: { tension: 300, friction: 20 },
  });

  // Rotation auto + réaction souris
  useFrame((state) => {
    if (!meshRef.current) return;

    const time = state.clock.getElapsedTime();

    // Rotation de base
    meshRef.current.rotation.x = Math.sin(time * 0.3) * 0.2;
    meshRef.current.rotation.y += 0.01;

    // Réaction au hover (suivre souris)
    if (hovered) {
      const mouse = state.mouse;
      meshRef.current.rotation.y += mouse.x * 0.02;
      meshRef.current.rotation.x += mouse.y * 0.02;
    }
  });

  return (
    <animated.mesh
      ref={meshRef}
      scale={scale}
      onPointerEnter={() => setHovered(true)}
      onPointerLeave={() => setHovered(false)}
    >
      {/* Géométrie icosahedron (low-poly) */}
      <icosahedronGeometry args={[1.5, 1]} />

      {/* Material metallic */}
      <meshStandardMaterial
        color={color}
        metalness={metalness}
        roughness={roughness}
        emissive={color}
        emissiveIntensity={hovered ? 0.3 : 0.1}
      />
    </animated.mesh>
  );
}

interface Avatar3DProps {
  color?: string;
  metalness?: number;
  roughness?: number;
  className?: string;
  height?: string;
  showFPS?: boolean;
}

/**
 * Composant Avatar 3D complet avec wrapper
 */
export function Avatar3D({
  color = '#3b82f6',
  metalness = 0.7,
  roughness = 0.3,
  className = '',
  height = '400px',
  showFPS = false,
}: Avatar3DProps) {
  return (
    <div className={className} style={{ height, width: '100%' }}>
      <Scene3DWrapper
        showFPS={showFPS}
        cameraPosition={[0, 0, 4]}
        fallback={
          <div className="flex items-center justify-center h-full bg-gradient-to-br from-blue-500 to-purple-500 rounded-lg">
            <div className="text-white text-center">
              <div className="text-6xl mb-4">👤</div>
              <p className="text-sm">Avatar 3D</p>
            </div>
          </div>
        }
      >
        <Avatar3DMesh color={color} metalness={metalness} roughness={roughness} />
      </Scene3DWrapper>
    </div>
  );
}

/**
 * Variante minimaliste (cube)
 */
export function AvatarCube3D({ color = '#8b5cf6', className = '', height = '300px' }) {
  const CubeMesh = () => {
    const meshRef = useRef<THREE.Mesh>(null);

    useFrame(() => {
      if (!meshRef.current) return;
      meshRef.current.rotation.x += 0.005;
      meshRef.current.rotation.y += 0.01;
    });

    return (
      <mesh ref={meshRef}>
        <boxGeometry args={[2, 2, 2]} />
        <meshStandardMaterial
          color={color}
          metalness={0.8}
          roughness={0.2}
          emissive={color}
          emissiveIntensity={0.2}
        />
      </mesh>
    );
  };

  return (
    <div className={className} style={{ height, width: '100%' }}>
      <Scene3DWrapper cameraPosition={[0, 0, 5]}>
        <CubeMesh />
      </Scene3DWrapper>
    </div>
  );
}

/**
 * Avatar avec plusieurs formes (demo)
 */
export function AvatarMultiShape3D({ className = '', height = '400px' }) {
  const MultiShapeMesh = () => {
    const groupRef = useRef<THREE.Group>(null);

    useFrame((state) => {
      if (!groupRef.current) return;
      const time = state.clock.getElapsedTime();
      groupRef.current.rotation.y = time * 0.5;
    });

    return (
      <group ref={groupRef}>
        {/* Sphère centrale */}
        <mesh position={[0, 0, 0]}>
          <sphereGeometry args={[1, 32, 32]} />
          <meshStandardMaterial color="#3b82f6" metalness={0.7} roughness={0.3} />
        </mesh>

        {/* Anneaux */}
        <mesh rotation={[Math.PI / 2, 0, 0]}>
          <torusGeometry args={[1.5, 0.1, 16, 100]} />
          <meshStandardMaterial color="#8b5cf6" metalness={0.8} roughness={0.2} />
        </mesh>

        <mesh rotation={[0, 0, Math.PI / 2]}>
          <torusGeometry args={[1.5, 0.1, 16, 100]} />
          <meshStandardMaterial color="#ec4899" metalness={0.8} roughness={0.2} />
        </mesh>
      </group>
    );
  };

  return (
    <div className={className} style={{ height, width: '100%' }}>
      <Scene3DWrapper cameraPosition={[0, 0, 5]}>
        <MultiShapeMesh />
      </Scene3DWrapper>
    </div>
  );
}
