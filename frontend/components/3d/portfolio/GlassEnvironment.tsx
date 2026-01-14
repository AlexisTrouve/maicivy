'use client';

import React from 'react';
import type { PerformanceLevel } from '@/lib/types';

interface GlassEnvironmentProps {
  performanceLevel: PerformanceLevel;
}

export function GlassEnvironment({ performanceLevel }: GlassEnvironmentProps) {
  const isHighPerf = performanceLevel === 'high';

  return (
    <>
      {/* Main key light - warm */}
      <spotLight
        position={[5, 8, 5]}
        angle={0.5}
        penumbra={1}
        intensity={2}
        color="#fff7ed"
      />

      {/* Fill light - cool */}
      <pointLight
        position={[-5, 3, -5]}
        intensity={1}
        color="#e0f2fe"
      />

      {/* Back light */}
      <directionalLight
        position={[0, 5, -10]}
        intensity={0.5}
        color="#c7d2fe"
      />

      {/* Ambient fill */}
      <ambientLight intensity={0.6} color="#ffffff" />
    </>
  );
}

export default GlassEnvironment;
