# Quick Start - Effets 3D

Guide rapide pour démarrer avec les composants 3D de maicivy.

---

## 🚀 Installation (5 min)

### Étape 1 : Installer les dépendances

```bash
cd /mnt/c/Users/alexi/Documents/projects/maicivy/frontend

npm install three@^0.160.0 \
  @react-three/fiber@^8.15.0 \
  @react-three/drei@^9.93.0 \
  @react-spring/three@^9.7.3

npm install -D @types/three@^0.160.0
```

**Vérification :**
```bash
npm list three @react-three/fiber @react-three/drei @react-spring/three
```

---

## 📦 Fichiers Créés

```
frontend/
├── components/3d/
│   ├── Avatar3D.tsx                    (188 lignes)
│   ├── SkillsGraph3D.tsx               (216 lignes)
│   ├── ParallaxBackground.tsx          (248 lignes)
│   ├── Scene3DWrapper.tsx              (143 lignes)
│   ├── index.ts                        (exports)
│   └── __tests__/Avatar3D.test.tsx     (136 lignes)
├── hooks/
│   ├── use3DSupport.ts                 (155 lignes)
│   └── use3DControls.ts                (112 lignes)
├── lib/
│   ├── 3d-utils.ts                     (395 lignes)
│   └── types.ts                        (modifié)
├── app/3d-demo/page.tsx                (278 lignes)
├── 3D_EFFECTS_IMPLEMENTATION_SUMMARY.md
└── QUICK_START_3D.md                   (ce fichier)
```

**Total : 1,953 lignes de code**

---

## 🎯 Test Rapide (2 min)

### 1. Démarrer le serveur

```bash
cd frontend
npm run dev
```

### 2. Visiter la page démo

Ouvrir dans le navigateur :
```
http://localhost:3000/3d-demo
```

**Vous devriez voir :**
- ✅ Avatar 3D Icosahedron (rotation interactive)
- ✅ Avatar Cube (rotation auto)
- ✅ Avatar Multi-Shapes (sphère + anneaux)
- ✅ Skills Graph 3D (8 compétences exemple)
- ✅ Controls Parallax Background
- ✅ Info WebGL (version, performance level)

### 3. Tester les interactions

**Avatar 3D :**
- Survolez avec la souris → Scale + Emissive augmentent
- Bougez la souris → Rotation suit le mouvement

**Skills Graph :**
- Cliquez sur une sphère → Selection + pulse animation
- Drag pour faire pivoter le graph
- Scroll pour zoomer

**Parallax Background :**
- Cliquez "Activer Parallax"
- Testez les 3 variants : stars, spiral, mixed
- Observez les particules et formes flottantes

---

## 🧪 Tests Unitaires (1 min)

```bash
npm run test components/3d/__tests__/Avatar3D.test.tsx
```

**Tests inclus :**
- ✅ Render Avatar3D
- ✅ Render AvatarCube3D
- ✅ Props customization
- ✅ WebGL fallback
- ✅ Performance variants

---

## 💡 Exemples d'Utilisation

### 1. Ajouter Avatar 3D à la homepage

**Fichier : `app/page.tsx`**

```tsx
import { Avatar3D } from '@/components/3d';

export default function HomePage() {
  return (
    <main>
      <section className="hero">
        <div className="container mx-auto px-4 py-12">
          {/* Avatar 3D */}
          <div className="max-w-md mx-auto mb-8">
            <Avatar3D height="300px" />
          </div>

          <h1 className="text-4xl font-bold text-center">
            Alexi - Développeur Full-Stack
          </h1>
          <p className="text-gray-600 text-center mt-4">
            Go • React • TypeScript • DevOps
          </p>
        </div>
      </section>
    </main>
  );
}
```

### 2. Ajouter Skills Graph à la page CV

**Fichier : `app/cv/page.tsx`**

```tsx
import { SkillsGraph3D } from '@/components/3d';

export default function CVPage() {
  // Récupérer skills depuis API
  const skills = [
    { id: '1', name: 'Go', level: 85, category: 'backend', yearsExperience: 3 },
    { id: '2', name: 'React', level: 95, category: 'frontend', yearsExperience: 5 },
    { id: '3', name: 'Docker', level: 85, category: 'devops', yearsExperience: 4 },
    // ...
  ];

  return (
    <main>
      <section className="skills">
        <h2 className="text-3xl font-bold mb-6">Mes Compétences</h2>

        {/* Skills Graph 3D */}
        <SkillsGraph3D
          skills={skills}
          autoRotate={true}
          height="600px"
          showFPS={false}
        />
      </section>
    </main>
  );
}
```

### 3. Ajouter Background Parallax

**Fichier : `app/layout.tsx`**

```tsx
import { MinimalBackground } from '@/components/3d';

export default function RootLayout({ children }) {
  return (
    <html lang="fr">
      <body>
        {/* Background 3D */}
        <MinimalBackground className="opacity-30" />

        {/* Contenu */}
        <div className="relative z-10">
          {children}
        </div>
      </body>
    </html>
  );
}
```

---

## 🎨 Customization

### Couleurs Avatar

```tsx
<Avatar3D
  color="#8b5cf6"      // Couleur principale
  metalness={0.8}      // 0-1 (plus métallique)
  roughness={0.2}      // 0-1 (plus lisse)
/>
```

### Variants Parallax

```tsx
// Étoiles classiques
<ParallaxBackground variant="stars" />

// Spirale
<ParallaxBackground variant="spiral" />

// Mix des deux
<ParallaxBackground variant="mixed" />
```

### Performance

```tsx
// Désactiver sur mobile
import { use3DSupport } from '@/hooks/use3DSupport';

const { isMobile } = use3DSupport();

{!isMobile && <Avatar3D />}
```

---

## 🔍 Vérification Support WebGL

**Dans la page démo (`/3d-demo`)**, en haut vous verrez :

```
WebGL Support
✓ WebGL 2 • Performance: high • Desktop
```

**Ou si non supporté :**

```
✗ WebGL not available
```

**Fallback automatique** → Composants affichent version 2D.

---

## 📊 Performance Monitoring

### Activer FPS Counter

```tsx
<Avatar3D showFPS={true} />
<SkillsGraph3D showFPS={true} />
<ParallaxBackground showFPS={true} />
```

**Affiche en haut à droite :**
```
60 FPS (high)
```

### Target FPS

- **Desktop High :** 60 FPS
- **Desktop Medium :** 45-60 FPS
- **Mobile High-end :** 30-45 FPS
- **Mobile Mid-range :** 30 FPS

---

## 🐛 Troubleshooting

### Erreur "Cannot find module 'three'"

```bash
npm install three @react-three/fiber @react-three/drei @react-spring/three
```

### FPS bas sur desktop

1. Vérifier GPU utilisé (integrated vs dedicated)
2. Réduire particules :
```tsx
<ParallaxBackground particleCount={300} />
```
3. Désactiver shadows :
```tsx
<Scene3DWrapper config={{ shadows: false }} />
```

### "Canvas is undefined"

Vérifier import :
```tsx
// ✅ Correct
'use client'; // En haut du fichier

// ❌ Si erreur SSR
import dynamic from 'next/dynamic';
const Avatar3D = dynamic(() => import('@/components/3d').then(m => m.Avatar3D), { ssr: false });
```

### Text drei ne s'affiche pas

Désactiver sur mobile low-end :
```tsx
import { use3DSupport } from '@/hooks/use3DSupport';

const { performanceLevel } = use3DSupport();

{performanceLevel !== 'low' && <Text>Label</Text>}
```

---

## 📚 Documentation Complète

- **`3D_EFFECTS_IMPLEMENTATION_SUMMARY.md`** - Documentation technique complète (600+ lignes)
- **`components/3d/README.md`** - Guide d'utilisation composants
- **Inline TSDoc** - Comments dans chaque fichier

---

## ✅ Checklist Intégration

### Installation
- [ ] Installer dépendances (`npm install three ...`)
- [ ] Vérifier installation (`npm list three`)

### Test
- [ ] Démarrer serveur (`npm run dev`)
- [ ] Visiter `/3d-demo`
- [ ] Vérifier WebGL support (badge en haut)
- [ ] Tester interactions (hover, click, drag)

### Intégration
- [ ] Ajouter `Avatar3D` à homepage
- [ ] Ajouter `SkillsGraph3D` à page CV
- [ ] (Optionnel) Ajouter `ParallaxBackground`

### Tests
- [ ] Run tests unitaires (`npm run test`)
- [ ] Tester sur mobile (DevTools responsive)
- [ ] Tester avec WebGL désactivé (fallback 2D)

### Production
- [ ] Vérifier bundle size (`npm run build`)
- [ ] Tester performances (FPS counter)
- [ ] Lazy loading si nécessaire

---

## 🚀 Prochaines Étapes

1. **Installer dépendances** (5 min)
2. **Tester page démo** (2 min)
3. **Intégrer dans pages existantes** (10 min)
4. **Customiser couleurs/variants** (5 min)
5. **Run tests** (1 min)
6. **Build production** (2 min)

**Total : ~25 minutes pour intégration complète**

---

## 💡 Commandes Utiles

```bash
# Installation
npm install three @react-three/fiber @react-three/drei @react-spring/three

# Dev
npm run dev

# Tests
npm run test components/3d/__tests__/Avatar3D.test.tsx

# Build
npm run build

# Type check
npx tsc --noEmit

# Lint
npm run lint
```

---

## 📞 Support

**Issues :**
- Vérifier `3D_EFFECTS_IMPLEMENTATION_SUMMARY.md` section "Troubleshooting"
- Checker `components/3d/README.md` section "🐛 Troubleshooting"

**Resources :**
- [Three.js Docs](https://threejs.org/docs/)
- [React Three Fiber](https://docs.pmnd.rs/react-three-fiber)
- [Drei Components](https://github.com/pmndrs/drei)

---

**Version :** 1.0
**Date :** 2025-12-08
**Ready to Use :** ✅ YES

🎨 **Enjoy the 3D effects!** 🚀
