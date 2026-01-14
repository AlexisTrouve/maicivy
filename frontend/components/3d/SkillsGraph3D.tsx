/**
 * Graph 3D des compétences
 * Nodes = skills (taille ∝ level), Edges = relations
 * Rotation auto, Click node → zoom + détails
 */

'use client';

import React, { useRef, useState, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { Text, Line } from '@react-three/drei';
import * as THREE from 'three';
import { Scene3DWrapper } from './Scene3DWrapper';
import { generateSkillsGraph } from '@/lib/3d-utils';
import type { SkillNode3D, SkillEdge3D, Skill } from '@/lib/types';

interface SkillNodeMeshProps {
  node: SkillNode3D;
  isSelected: boolean;
  onSelect: () => void;
}

/**
 * Node individuel (skill)
 */
function SkillNodeMesh({ node, isSelected, onSelect }: SkillNodeMeshProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);

  useFrame(() => {
    if (!meshRef.current) return;

    // Rotation douce
    meshRef.current.rotation.y += 0.002;

    // Scale animation si selected
    if (isSelected) {
      const scale = 1.2 + Math.sin(Date.now() * 0.003) * 0.1;
      meshRef.current.scale.setScalar(scale);
    } else {
      meshRef.current.scale.setScalar(1);
    }
  });

  return (
    <group position={node.position}>
      <mesh
        ref={meshRef}
        onClick={onSelect}
        onPointerEnter={() => setHovered(true)}
        onPointerLeave={() => setHovered(false)}
      >
        <sphereGeometry args={[node.radius, 32, 32]} />
        <meshStandardMaterial
          color={node.color}
          metalness={0.6}
          roughness={0.4}
          emissive={node.color}
          emissiveIntensity={hovered || isSelected ? 0.5 : 0.2}
        />
      </mesh>

      {/* Label */}
      {(hovered || isSelected) && (
        <Text
          position={[0, node.radius + 0.5, 0]}
          fontSize={0.3}
          color="white"
          anchorX="center"
          anchorY="middle"
          outlineWidth={0.05}
          outlineColor="#000000"
        >
          {node.name}
        </Text>
      )}
    </group>
  );
}

interface EdgeLineProps {
  start: [number, number, number];
  end: [number, number, number];
  strength: number;
}

/**
 * Ligne reliant deux nodes
 */
function EdgeLine({ start, end, strength }: EdgeLineProps) {
  const points = useMemo(() => {
    return [start, end] as [
      [number, number, number],
      [number, number, number]
    ];
  }, [start, end]);

  return (
    <Line
      points={points}
      color="#6366f1"
      lineWidth={2}
      opacity={strength * 0.5}
      transparent
    />
  );
}

interface SkillsGraph3DSceneProps {
  nodes: SkillNode3D[];
  edges: SkillEdge3D[];
  autoRotate?: boolean;
}

/**
 * Scène du graph (nodes + edges)
 */
function SkillsGraph3DScene({ nodes, edges, autoRotate = true }: SkillsGraph3DSceneProps) {
  const groupRef = useRef<THREE.Group>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  // Auto-rotation
  useFrame(() => {
    if (!groupRef.current || !autoRotate) return;
    groupRef.current.rotation.y += 0.003;
  });

  return (
    <group ref={groupRef}>
      {/* Edges (lignes) */}
      {edges.map((edge, i) => {
        const sourceNode = nodes.find((n) => n.id === edge.source);
        const targetNode = nodes.find((n) => n.id === edge.target);

        if (!sourceNode || !targetNode) return null;

        return (
          <EdgeLine
            key={`edge-${i}`}
            start={sourceNode.position}
            end={targetNode.position}
            strength={edge.strength}
          />
        );
      })}

      {/* Nodes (skills) */}
      {nodes.map((node) => (
        <SkillNodeMesh
          key={node.id}
          node={node}
          isSelected={selectedNode === node.id}
          onSelect={() => setSelectedNode(node.id === selectedNode ? null : node.id)}
        />
      ))}
    </group>
  );
}

interface SkillsGraph3DProps {
  skills: Skill[];
  autoRotate?: boolean;
  className?: string;
  height?: string;
  showFPS?: boolean;
}

/**
 * Composant principal SkillsGraph3D
 */
export function SkillsGraph3D({
  skills,
  autoRotate = true,
  className = '',
  height = '600px',
  showFPS = false,
}: SkillsGraph3DProps) {
  // Générer le graph à partir des skills
  const { nodes, edges } = useMemo(() => {
    const skillsData = skills.map((s) => ({
      name: s.name,
      level: s.level,
      category: s.category,
    }));
    return generateSkillsGraph(skillsData);
  }, [skills]);

  return (
    <div className={className} style={{ height, width: '100%' }}>
      <Scene3DWrapper
        showFPS={showFPS}
        cameraPosition={[0, 0, 8]}
        fallback={
          <div className="flex items-center justify-center h-full bg-gradient-to-br from-indigo-500 to-purple-500 rounded-lg">
            <div className="text-white text-center p-8">
              <div className="text-6xl mb-4">🕸️</div>
              <h3 className="text-xl font-bold mb-2">Skills Graph 3D</h3>
              <p className="text-sm opacity-80">{skills.length} compétences</p>
            </div>
          </div>
        }
      >
        <SkillsGraph3DScene nodes={nodes} edges={edges} autoRotate={autoRotate} />
      </Scene3DWrapper>

      {/* Légende */}
      <div className="mt-4 text-sm text-gray-600">
        <p>💡 Cliquez sur une sphère pour la sélectionner</p>
        <p>💡 Utilisez la souris pour faire pivoter le graph</p>
      </div>
    </div>
  );
}

/**
 * Version simplifiée avec données exemple
 */
export function SkillsGraph3DDemo({ className = '', height = '500px' }) {
  const demoSkills: Skill[] = [
    { id: '1', name: 'Go', level: 85, category: 'backend', yearsExperience: 3 },
    { id: '2', name: 'TypeScript', level: 90, category: 'frontend', yearsExperience: 5 },
    { id: '3', name: 'React', level: 95, category: 'frontend', yearsExperience: 5 },
    { id: '4', name: 'PostgreSQL', level: 80, category: 'database', yearsExperience: 4 },
    { id: '5', name: 'Docker', level: 85, category: 'devops', yearsExperience: 4 },
    { id: '6', name: 'AWS', level: 70, category: 'cloud', yearsExperience: 2 },
    { id: '7', name: 'Node.js', level: 88, category: 'backend', yearsExperience: 5 },
    { id: '8', name: 'Next.js', level: 92, category: 'frontend', yearsExperience: 3 },
  ];

  return <SkillsGraph3D skills={demoSkills} className={className} height={height} />;
}
