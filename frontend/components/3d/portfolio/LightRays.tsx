'use client';

import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface LightRaysProps {
  count?: number;
  color?: string;
  intensity?: number;
}

export function LightRays({
  count = 8,
  color = '#8b5cf6',
  intensity = 0.2
}: LightRaysProps) {
  const groupRef = useRef<THREE.Group>(null);
  const raysRef = useRef<THREE.Mesh[]>([]);

  // Create ray configurations
  const rayConfigs = useMemo(() => {
    return Array.from({ length: count }, (_, i) => {
      const angle = (i / count) * Math.PI * 2;
      const radius = 6 + Math.random() * 2;
      return {
        x: Math.cos(angle) * radius,
        z: Math.sin(angle) * radius,
        angle,
        height: 12 + Math.random() * 8,
        width: 0.3 + Math.random() * 0.4,
        speed: 0.3 + Math.random() * 0.4,
        phase: Math.random() * Math.PI * 2
      };
    });
  }, [count]);

  // Animate rays
  useFrame((state) => {
    if (!groupRef.current) return;

    const time = state.clock.getElapsedTime();

    raysRef.current.forEach((ray, i) => {
      if (ray && ray.material) {
        const config = rayConfigs[i];
        // Gentle pulsing
        const pulse = Math.sin(time * config.speed + config.phase) * 0.5 + 0.5;
        const mat = ray.material as THREE.MeshBasicMaterial;
        mat.opacity = intensity * (0.3 + pulse * 0.7);
      }
    });

    // Very slow overall rotation
    groupRef.current.rotation.y = time * 0.01;
  });

  return (
    <group ref={groupRef} position={[0, 0, 0]}>
      {rayConfigs.map((config, i) => (
        <mesh
          key={i}
          ref={(el) => { if (el) raysRef.current[i] = el; }}
          position={[config.x, config.height / 2, config.z]}
          rotation={[0, 0, 0]}
        >
          {/* Smooth cylinder for light beam effect */}
          <cylinderGeometry args={[config.width, config.width * 2, config.height, 16, 1, true]} />
          <meshBasicMaterial
            color={color}
            transparent
            opacity={intensity}
            side={THREE.DoubleSide}
            blending={THREE.AdditiveBlending}
            depthWrite={false}
          />
        </mesh>
      ))}

      {/* Central glow */}
      <mesh position={[0, 8, 0]}>
        <sphereGeometry args={[1.5, 16, 16]} />
        <meshBasicMaterial
          color={color}
          transparent
          opacity={intensity * 0.5}
          blending={THREE.AdditiveBlending}
          depthWrite={false}
        />
      </mesh>
    </group>
  );
}

export default LightRays;
