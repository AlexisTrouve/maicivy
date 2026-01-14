/**
 * Utilitaires pour les scènes 3D
 * Helpers Three.js, génération de données, optimisations
 */

import * as THREE from 'three';

// Types
export interface SkillNode3D {
  id: string;
  name: string;
  level: number; // 0-1
  category: string;
  color: string;
  position: [number, number, number];
  radius: number;
}

export interface SkillEdge3D {
  source: string; // node id
  target: string; // node id
  strength: number; // 0-1
}

export interface Scene3DConfig {
  antialias?: boolean;
  shadows?: boolean;
  pixelRatio?: number;
  alpha?: boolean;
  powerPreference?: 'high-performance' | 'low-power' | 'default';
}

/**
 * Catégories de skills avec couleurs prédéfinies
 */
export const SKILL_CATEGORIES = {
  backend: '#3b82f6', // blue-500
  frontend: '#8b5cf6', // purple-500
  devops: '#10b981', // green-500
  database: '#f59e0b', // amber-500
  cloud: '#06b6d4', // cyan-500
  tools: '#6366f1', // indigo-500
  languages: '#ec4899', // pink-500
  other: '#6b7280', // gray-500
} as const;

/**
 * Relations prédéfinies entre catégories de skills
 */
const CATEGORY_RELATIONS = {
  backend: ['database', 'devops', 'cloud'],
  frontend: ['backend', 'tools'],
  devops: ['backend', 'cloud'],
  database: ['backend', 'cloud'],
  cloud: ['devops', 'database'],
  tools: ['frontend', 'backend'],
  languages: ['backend', 'frontend'],
};

/**
 * Génère un graph 3D de skills à partir d'une liste de compétences
 */
export function generateSkillsGraph(
  skills: Array<{ name: string; level: number; category: string }>
): { nodes: SkillNode3D[]; edges: SkillEdge3D[] } {
  const nodes: SkillNode3D[] = [];
  const edges: SkillEdge3D[] = [];

  // 1. Créer les nodes
  skills.forEach((skill, index) => {
    const category = skill.category.toLowerCase();
    const color = SKILL_CATEGORIES[category as keyof typeof SKILL_CATEGORIES] || SKILL_CATEGORIES.other;

    // Position en sphère de Fibonacci
    const position = fibonacciSpherePoint(index, skills.length, 3);

    nodes.push({
      id: `skill_${index}`,
      name: skill.name,
      level: skill.level / 100, // Normaliser 0-100 → 0-1
      category,
      color,
      position,
      radius: 0.2 + (skill.level / 100) * 0.3, // Taille ∝ level
    });
  });

  // 2. Créer les edges (relations entre catégories)
  for (let i = 0; i < nodes.length; i++) {
    const nodeA = nodes[i];
    const relatedCategories = CATEGORY_RELATIONS[nodeA.category as keyof typeof CATEGORY_RELATIONS] || [];

    for (let j = i + 1; j < nodes.length; j++) {
      const nodeB = nodes[j];

      // Si les catégories sont liées
      if (relatedCategories.includes(nodeB.category)) {
        edges.push({
          source: nodeA.id,
          target: nodeB.id,
          strength: Math.min(nodeA.level, nodeB.level),
        });
      }
    }
  }

  return { nodes, edges };
}

/**
 * Distribution de points sur une sphère (Fibonacci sphere)
 * Optimal pour répartir N points uniformément
 */
export function fibonacciSpherePoint(
  index: number,
  total: number,
  radius: number = 1
): [number, number, number] {
  const phi = Math.PI * (3 - Math.sqrt(5)); // Golden angle
  const y = 1 - (index / (total - 1)) * 2; // y de 1 à -1
  const radiusAtY = Math.sqrt(1 - y * y);
  const theta = phi * index;

  const x = Math.cos(theta) * radiusAtY * radius;
  const z = Math.sin(theta) * radiusAtY * radius;

  return [x, y * radius, z];
}

/**
 * Génère un gradient de couleurs entre deux couleurs
 */
export function generateColorGradient(
  color1: string,
  color2: string,
  steps: number
): string[] {
  const c1 = new THREE.Color(color1);
  const c2 = new THREE.Color(color2);

  const colors: string[] = [];
  for (let i = 0; i < steps; i++) {
    const t = i / (steps - 1);
    const color = new THREE.Color().lerpColors(c1, c2, t);
    colors.push(`#${color.getHexString()}`);
  }

  return colors;
}

/**
 * Génère des positions aléatoires pour des particules dans une boîte
 */
export function generateParticlePositions(
  count: number,
  bounds: { x: number; y: number; z: number } = { x: 10, y: 10, z: 10 }
): Float32Array {
  const positions = new Float32Array(count * 3);

  for (let i = 0; i < count; i++) {
    positions[i * 3] = (Math.random() - 0.5) * bounds.x;
    positions[i * 3 + 1] = (Math.random() - 0.5) * bounds.y;
    positions[i * 3 + 2] = (Math.random() - 0.5) * bounds.z;
  }

  return positions;
}

/**
 * Génère des positions en spirale
 */
export function generateSpiralPositions(
  count: number,
  radius: number = 5,
  height: number = 10
): Float32Array {
  const positions = new Float32Array(count * 3);
  const turns = 3;

  for (let i = 0; i < count; i++) {
    const t = i / count;
    const angle = t * Math.PI * 2 * turns;
    const r = radius * (1 - t * 0.5);
    const y = (t - 0.5) * height;

    positions[i * 3] = Math.cos(angle) * r;
    positions[i * 3 + 1] = y;
    positions[i * 3 + 2] = Math.sin(angle) * r;
  }

  return positions;
}

/**
 * Crée une géométrie optimisée pour instanced rendering
 */
export function createInstancedGeometry(
  geometry: THREE.BufferGeometry,
  positions: Float32Array,
  colors?: Float32Array,
  scales?: Float32Array
): THREE.InstancedBufferGeometry {
  const instancedGeometry = new THREE.InstancedBufferGeometry();

  // Copier attributs de base
  instancedGeometry.index = geometry.index;
  instancedGeometry.attributes = geometry.attributes;

  // Ajouter positions instances
  const instancePositions = new THREE.InstancedBufferAttribute(positions, 3);
  instancedGeometry.setAttribute('instancePosition', instancePositions);

  // Ajouter couleurs si fournies
  if (colors) {
    const instanceColors = new THREE.InstancedBufferAttribute(colors, 3);
    instancedGeometry.setAttribute('instanceColor', instanceColors);
  }

  // Ajouter scales si fournies
  if (scales) {
    const instanceScales = new THREE.InstancedBufferAttribute(scales, 1);
    instancedGeometry.setAttribute('instanceScale', instanceScales);
  }

  return instancedGeometry;
}

/**
 * Calcule les FPS moyens
 */
export class FPSMonitor {
  private frames: number = 0;
  private lastTime: number = performance.now();
  private fps: number = 60;

  update(): number {
    this.frames++;
    const now = performance.now();
    const delta = now - this.lastTime;

    if (delta >= 1000) {
      this.fps = Math.round((this.frames * 1000) / delta);
      this.frames = 0;
      this.lastTime = now;
    }

    return this.fps;
  }

  getFPS(): number {
    return this.fps;
  }
}

/**
 * Optimise un material pour performance
 */
export function optimizeMaterial(material: THREE.Material): void {
  // Désactiver shadows si pas nécessaire
  (material as any).shadowSide = THREE.FrontSide;

  // Frustum culling
  (material as any).side = THREE.FrontSide;

  // Flatshading pour géométries simples
  if (material instanceof THREE.MeshStandardMaterial) {
    material.flatShading = false;
  }
}

/**
 * Dispose proprement un objet 3D (libération mémoire)
 */
export function disposeObject3D(object: THREE.Object3D): void {
  object.traverse((child) => {
    if (child instanceof THREE.Mesh) {
      if (child.geometry) {
        child.geometry.dispose();
      }

      if (child.material) {
        if (Array.isArray(child.material)) {
          child.material.forEach((mat) => mat.dispose());
        } else {
          child.material.dispose();
        }
      }
    }
  });
}

/**
 * Calcule la distance camera optimale selon taille objet
 */
export function calculateOptimalCameraDistance(
  boundingBox: THREE.Box3,
  fov: number = 75
): number {
  const size = boundingBox.getSize(new THREE.Vector3());
  const maxDim = Math.max(size.x, size.y, size.z);

  const cameraDistance = maxDim / (2 * Math.tan((fov * Math.PI) / 360));

  return cameraDistance * 1.5; // Marge
}

/**
 * Crée un material standard optimisé
 */
export function createOptimizedMaterial(
  color: string | number,
  options: {
    metalness?: number;
    roughness?: number;
    emissive?: string | number;
    emissiveIntensity?: number;
  } = {}
): THREE.MeshStandardMaterial {
  const material = new THREE.MeshStandardMaterial({
    color,
    metalness: options.metalness ?? 0.5,
    roughness: options.roughness ?? 0.5,
    emissive: options.emissive ?? 0x000000,
    emissiveIntensity: options.emissiveIntensity ?? 0,
  });

  optimizeMaterial(material);

  return material;
}

/**
 * Lerp (linear interpolation) pour valeurs numériques
 */
export function lerp(start: number, end: number, t: number): number {
  return start + (end - start) * t;
}

/**
 * Clamp une valeur entre min et max
 */
export function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

// ============================================
// Portfolio 3D Layout Functions
// ============================================

export interface CardPosition3D {
  position: [number, number, number];
  rotation: [number, number, number];
  scale: number;
}

/**
 * Generate circular layout for portfolio cards
 * Cards arranged in a circle, facing the center
 */
export function generateCircularLayout(
  count: number,
  radius: number = 4,
  yOffset: number = 0
): CardPosition3D[] {
  return Array.from({ length: count }, (_, i) => {
    const angle = (i / count) * Math.PI * 2;
    return {
      position: [
        Math.sin(angle) * radius,
        yOffset,
        Math.cos(angle) * radius
      ] as [number, number, number],
      rotation: [0, angle + Math.PI, 0] as [number, number, number], // Face center
      scale: 1
    };
  });
}

/**
 * Generate spiral/helix layout for portfolio cards
 * Good for many projects
 */
export function generateSpiralLayout(
  count: number,
  baseRadius: number = 3,
  heightRange: number = 4,
  turns: number = 1.5
): CardPosition3D[] {
  return Array.from({ length: count }, (_, i) => {
    const t = i / (count - 1 || 1);
    const angle = t * Math.PI * 2 * turns;
    const radius = baseRadius + t * 1.5; // Expanding spiral
    const y = (t - 0.5) * heightRange;

    return {
      position: [
        Math.sin(angle) * radius,
        y,
        Math.cos(angle) * radius
      ] as [number, number, number],
      rotation: [0, angle + Math.PI, 0] as [number, number, number],
      scale: 1 - t * 0.2 // Slightly smaller as they go up
    };
  });
}

/**
 * Generate grid layout for portfolio cards
 * Simple flat grid arrangement
 */
export function generateGridLayout(
  count: number,
  columns: number = 3,
  spacing: number = 2.5
): CardPosition3D[] {
  const rows = Math.ceil(count / columns);
  const offsetX = ((columns - 1) * spacing) / 2;
  const offsetY = ((rows - 1) * spacing) / 2;

  return Array.from({ length: count }, (_, i) => {
    const col = i % columns;
    const row = Math.floor(i / columns);

    return {
      position: [
        col * spacing - offsetX,
        row * -spacing + offsetY,
        0
      ] as [number, number, number],
      rotation: [0, 0, 0] as [number, number, number],
      scale: 1
    };
  });
}

/**
 * Smooth camera transition helper
 * Returns interpolated position/target for smooth camera movement
 */
export function smoothCameraTransition(
  currentPos: THREE.Vector3,
  targetPos: THREE.Vector3,
  currentLookAt: THREE.Vector3,
  targetLookAt: THREE.Vector3,
  smoothness: number = 0.05
): { position: THREE.Vector3; lookAt: THREE.Vector3 } {
  return {
    position: currentPos.clone().lerp(targetPos, smoothness),
    lookAt: currentLookAt.clone().lerp(targetLookAt, smoothness)
  };
}
