# Composants 3D - maicivy

**Feature 4 : Effets 3D Optionnels**

Ce dossier contient tous les composants React Three Fiber pour les visualisations 3D interactives du projet.

---

## 📦 Composants Disponibles

### Scene3DWrapper

Wrapper principal pour toutes les scènes 3D. Gère Canvas, Camera, Lights, Controls.

```tsx
import { Scene3DWrapper } from '@/components/3d';

<Scene3DWrapper
  showFPS={true}
  cameraPosition={[0, 0, 5]}
  enableControls={true}
  fallback={<CustomFallback />}
>
  {/* Contenu 3D */}
</Scene3DWrapper>
```

**Props:**
- `config?: Scene3DConfig` - Configuration Canvas (antialias, shadows, etc.)
- `showFPS?: boolean` - Afficher compteur FPS
- `fallback?: ReactNode` - Fallback si WebGL non supporté
- `cameraPosition?: [x, y, z]` - Position initiale camera
- `enableControls?: boolean` - Activer OrbitControls

---

### Avatar3D

Avatar 3D interactif avec géométrie low-poly (icosahedron).

```tsx
import { Avatar3D } from '@/components/3d';

<Avatar3D
  color="#3b82f6"
  metalness={0.7}
  roughness={0.3}
  height="400px"
  showFPS={false}
/>
```

**Variantes:**
- `Avatar3D` - Icosahedron principal
- `AvatarCube3D` - Cube simple
- `AvatarMultiShape3D` - Sphère + anneaux

**Features:**
- Rotation automatique
- Hover interaction (scale + emissive)
- Suit le mouvement de la souris

---

### SkillsGraph3D

Graph 3D des compétences avec nodes et edges.

```tsx
import { SkillsGraph3D } from '@/components/3d';

const skills = [
  { id: '1', name: 'Go', level: 85, category: 'backend', yearsExperience: 3 },
  { id: '2', name: 'React', level: 95, category: 'frontend', yearsExperience: 5 },
  // ...
];

<SkillsGraph3D
  skills={skills}
  autoRotate={true}
  height="600px"
  showFPS={false}
/>
```

**Features:**
- Nodes = sphères (taille ∝ level, couleur par catégorie)
- Edges = lignes reliant catégories liées
- Click node → selection + pulse animation
- Hover → label apparaît
- Auto-rotation

**Variante:**
- `SkillsGraph3DDemo` - Avec données exemple

---

### ParallaxBackground

Background 3D avec particules et parallax.

```tsx
import { ParallaxBackground } from '@/components/3d';

<ParallaxBackground
  variant="stars" // ou "spiral" ou "mixed"
  showShapes={true}
  height="100vh"
/>
```

**Variants:**
- `stars` - Particules aléatoires (étoiles)
- `spiral` - Particules en spirale
- `mixed` - Mix des deux

**Variantes Composant:**
- `ParallaxBackground` - Full screen fixed
- `ParallaxOverlay` - Overlay avec opacity
- `MinimalBackground` - Version optimisée (300 particules max)

---

## 🎣 Hooks Disponibles

### use3DSupport

Détecte le support WebGL et les performances du device.

```tsx
import { use3DSupport } from '@/hooks/use3DSupport';

const { isSupported, performanceLevel, webGLVersion, isMobile, reason } = use3DSupport();

if (!isSupported) {
  return <Fallback reason={reason} />;
}
```

**Returns:**
```typescript
{
  isSupported: boolean;
  performanceLevel: 'high' | 'medium' | 'low' | 'none';
  webGLVersion: 1 | 2 | null;
  isMobile: boolean;
  reason?: string; // Si non supporté
}
```

### use3DQualitySettings

Retourne les settings de qualité selon performance.

```tsx
import { use3DQualitySettings } from '@/hooks/use3DSupport';

const settings = use3DQualitySettings();
// { antialias, shadows, particleCount, maxFPS, pixelRatio }

<ParallaxBackground particleCount={settings.particleCount} />
```

### use3DControls

Configure OrbitControls avec smooth damping.

```tsx
import { use3DControls } from '@/hooks/use3DControls';

const controlsRef = useRef();

use3DControls(controlsRef, {
  enableDamping: true,
  dampingFactor: 0.05,
  autoRotate: false,
  minDistance: 2,
  maxDistance: 10
});

<OrbitControls ref={controlsRef} />
```

---

## 🛠️ Utilitaires

### Génération Graph

```tsx
import { generateSkillsGraph } from '@/lib/3d-utils';

const skills = [
  { name: 'Go', level: 85, category: 'backend' },
  { name: 'React', level: 95, category: 'frontend' }
];

const { nodes, edges } = generateSkillsGraph(skills);
// nodes: SkillNode3D[]
// edges: SkillEdge3D[]
```

### Positionnement

```tsx
import {
  fibonacciSpherePoint,
  generateParticlePositions,
  generateSpiralPositions
} from '@/lib/3d-utils';

// Distribution uniforme sur sphère
const position = fibonacciSpherePoint(index, total, radius);

// Particules aléatoires dans box
const positions = generateParticlePositions(1000, { x: 10, y: 10, z: 10 });

// Particules en spirale
const spiralPositions = generateSpiralPositions(500, radius, height);
```

### Optimisations

```tsx
import {
  optimizeMaterial,
  disposeObject3D,
  FPSMonitor
} from '@/lib/3d-utils';

// Optimiser material
optimizeMaterial(material);

// Libérer mémoire
disposeObject3D(object3D);

// Monitor FPS
const monitor = new FPSMonitor();
const fps = monitor.update(); // Dans useFrame
```

---

## 🎨 Catégories Skills & Couleurs

```typescript
import { SKILL_CATEGORIES } from '@/lib/3d-utils';

// Couleurs prédéfinies
backend: '#3b82f6',    // blue-500
frontend: '#8b5cf6',   // purple-500
devops: '#10b981',     // green-500
database: '#f59e0b',   // amber-500
cloud: '#06b6d4',      // cyan-500
tools: '#6366f1',      // indigo-500
languages: '#ec4899',  // pink-500
other: '#6b7280'       // gray-500
```

---

## 📱 Responsive & Performance

### Détection Automatique

Les composants détectent automatiquement:
- Support WebGL (v1, v2, none)
- Performance device (high, medium, low)
- Mobile vs Desktop
- GPU (NVIDIA, Radeon, Intel, Apple M1/M2)

### Adaptation Qualité

**High Performance (Desktop GPU performant):**
- 1000 particules
- 60 FPS target
- Antialiasing ON
- Shadows ON
- PixelRatio: 2

**Medium Performance (Desktop mid-range):**
- 500 particules
- 45 FPS target
- Antialiasing ON
- Shadows OFF
- PixelRatio: 1

**Low Performance (Mobile):**
- 200 particules
- 30 FPS target
- Antialiasing OFF
- Shadows OFF
- PixelRatio: 1

**None (WebGL non supporté):**
- Fallback 2D automatique

---

## ⚡ Optimisations Appliquées

### Instanced Rendering

Particules utilisant `<Points>` + `InstancedBufferGeometry`:
- 1 draw call pour 1000 particules
- vs 1000 draw calls individuels

### Frustum Culling

Automatique Three.js:
- Objets hors champ non rendus
- Améliore performances

### Material Optimization

```typescript
material.shadowSide = THREE.FrontSide;
material.side = THREE.FrontSide;
```

### Memory Cleanup

Toujours dispose des objets au unmount:

```tsx
useEffect(() => {
  return () => {
    disposeObject3D(scene);
  };
}, []);
```

---

## 🧪 Tests

### Tests Unitaires

```bash
npm run test components/3d/__tests__/Avatar3D.test.tsx
```

**Mocks inclus:**
- `@react-three/fiber`
- `@react-three/drei`
- `@react-spring/three`
- Hooks 3D

### Test Manuel

Page démo: `/3d-demo`

```bash
npm run dev
# Visiter http://localhost:3000/3d-demo
```

---

## 📦 Installation Dépendances

```bash
npm install three @react-three/fiber @react-three/drei @react-spring/three
npm install -D @types/three
```

---

## 🐛 Troubleshooting

### "Canvas is undefined"

Vérifier import:
```tsx
// ✅ Correct
import { Canvas } from '@react-three/fiber';

// ❌ Incorrect
import { Canvas } from 'three';
```

### FPS bas sur desktop

Vérifier:
1. GPU utilisé (integrated vs dedicated)
2. Nombre de particules (réduire si nécessaire)
3. Chrome hardware acceleration activée

### Text drei ne s'affiche pas

Désactiver Text sur mobile low-end:
```tsx
{performanceLevel !== 'low' && <Text>Label</Text>}
```

### Memory leak

Toujours dispose:
```tsx
useEffect(() => {
  return () => {
    geometry.dispose();
    material.dispose();
  };
}, []);
```

---

## 📚 Références

- [Three.js Docs](https://threejs.org/docs/)
- [React Three Fiber](https://docs.pmnd.rs/react-three-fiber)
- [Drei Components](https://github.com/pmndrs/drei)
- [React Spring](https://www.react-spring.dev/)

---

## 💡 Exemples d'Intégration

### Page d'Accueil

```tsx
import { Avatar3D, MinimalBackground } from '@/components/3d';

export default function HomePage() {
  return (
    <>
      <MinimalBackground />
      <main className="relative z-10">
        <section className="hero">
          <Avatar3D height="300px" />
          <h1>Alexi - Développeur Full-Stack</h1>
        </section>
      </main>
    </>
  );
}
```

### Page CV

```tsx
import { SkillsGraph3D } from '@/components/3d';

export default function CVPage({ skills }) {
  return (
    <main>
      <section className="skills">
        <h2>Mes Compétences</h2>
        <SkillsGraph3D skills={skills} height="600px" />
      </section>
    </main>
  );
}
```

### Background Parallax

```tsx
import { ParallaxBackground } from '@/components/3d';

export default function Layout({ children }) {
  return (
    <>
      <ParallaxBackground variant="stars" />
      <div className="relative z-10">
        {children}
      </div>
    </>
  );
}
```

---

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Alexi
