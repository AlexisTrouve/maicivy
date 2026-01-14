'use client';

import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface AmbientParticlesProps {
  count?: number;
  spread?: number;
  size?: number;
  color?: string;
  speed?: number;
}

export function AmbientParticles({
  count = 300,
  spread = 15,
  size = 0.015,
  color = '#c7d2fe',
  speed = 0.2
}: AmbientParticlesProps) {
  const pointsRef = useRef<THREE.Points>(null);

  // Generate initial particle data - fixed size, never changes
  const { initialPositions, velocities } = useMemo(() => {
    const positions = new Float32Array(count * 3);
    const vels = new Float32Array(count * 3);

    for (let i = 0; i < count; i++) {
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      const r = Math.random() * spread;

      positions[i * 3] = r * Math.sin(phi) * Math.cos(theta);
      positions[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta);
      positions[i * 3 + 2] = r * Math.cos(phi);

      vels[i * 3] = (Math.random() - 0.5) * 0.01;
      vels[i * 3 + 1] = (Math.random() - 0.5) * 0.01;
      vels[i * 3 + 2] = (Math.random() - 0.5) * 0.01;
    }

    return { initialPositions: positions, velocities: vels };
  }, [count, spread]);

  // Create geometry once
  const geometry = useMemo(() => {
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(initialPositions.slice(), 3));
    return geo;
  }, [initialPositions]);

  // Animate particles
  useFrame((state) => {
    if (!pointsRef.current) return;

    const posAttr = pointsRef.current.geometry.attributes.position;
    const positions = posAttr.array as Float32Array;
    const time = state.clock.getElapsedTime() * speed;

    for (let i = 0; i < count; i++) {
      const i3 = i * 3;

      // Gentle drift
      positions[i3] += velocities[i3];
      positions[i3 + 1] += velocities[i3 + 1];
      positions[i3 + 2] += velocities[i3 + 2];

      // Add sine wave motion
      positions[i3 + 1] += Math.sin(time + i * 0.1) * 0.001;

      // Wrap around bounds
      if (Math.abs(positions[i3]) > spread) positions[i3] *= -0.9;
      if (Math.abs(positions[i3 + 1]) > spread) positions[i3 + 1] *= -0.9;
      if (Math.abs(positions[i3 + 2]) > spread) positions[i3 + 2] *= -0.9;
    }

    posAttr.needsUpdate = true;
  });

  return (
    <points ref={pointsRef} geometry={geometry} frustumCulled={true}>
      <pointsMaterial
        transparent
        color={color}
        size={size}
        sizeAttenuation={true}
        depthWrite={false}
        opacity={0.6}
        blending={THREE.AdditiveBlending}
      />
    </points>
  );
}

export default AmbientParticles;
