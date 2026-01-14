# 07. FRONTEND_CV_DYNAMIC - Interface CV Dynamique

## 📋 Métadonnées

- **Phase:** 2
- **Priorité:** HAUTE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 05. FRONTEND_FOUNDATION, 06. BACKEND_CV_API
- **Temps estimé:** 4-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Créer une interface interactive et animée pour afficher un CV dynamique qui s'adapte selon le thème sélectionné (backend, frontend, DevOps, etc.). L'interface permet de :

- Sélectionner un thème via query params ou interface visuelle
- Afficher les expériences, compétences et projets filtrés et scorés selon le thème
- Exporter le CV en PDF
- Offrir une expérience utilisateur fluide avec animations et design responsive

**Valeur ajoutée:**
- Démontre la capacité à créer des interfaces modernes et performantes
- Showcase technique des compétences frontend (Next.js, TypeScript, Framer Motion)
- Permet aux recruteurs de voir instantanément le profil adapté à leurs besoins

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────┐
│              Page /cv?theme=backend                  │
├─────────────────────────────────────────────────────┤
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │       CVThemeSelector Component               │  │
│  │  - Dropdown thèmes disponibles                │  │
│  │  - Preview du thème sélectionné               │  │
│  │  - Update query params                        │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │     ExperienceTimeline Component              │  │
│  │  - Timeline verticale des expériences         │  │
│  │  - Animations Framer Motion (scroll reveal)   │  │
│  │  - Filtrées selon le thème                    │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │        SkillsCloud Component                  │  │
│  │  - Tag cloud interactif                       │  │
│  │  - Taille = niveau de compétence              │  │
│  │  - Hover effects                              │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │       ProjectsGrid Component                  │  │
│  │  - Grid responsive de cards projets           │  │
│  │  - GitHub stars, languages                    │  │
│  │  - Links vers repos                           │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │         Export PDF Button                     │  │
│  │  - Téléchargement via API /api/cv/export     │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
└─────────────────────────────────────────────────────┘
```

### Flux de Données

```
1. User visite /cv?theme=backend
   ↓
2. Page extraits query param "theme"
   ↓
3. Fetch API GET /api/cv?theme=backend
   ↓
4. API retourne CV filtré et scoré
   ↓
5. Page affiche components avec données
   ↓
6. User change thème via CVThemeSelector
   ↓
7. Update query param → re-fetch → re-render
```

### Design Decisions

**Choix Techniques:**

1. **Next.js App Router:**
   - Server Components pour SEO optimal
   - Client Components pour interactivité
   - Suspense boundaries pour loading states

2. **Framer Motion:**
   - Animations fluides et performantes
   - Variants pour cohérence
   - Scroll-triggered animations (IntersectionObserver)

3. **Query Params:**
   - URL shareable (/cv?theme=backend)
   - Permet partage direct d'un CV thématique
   - SEO-friendly

4. **Responsive Design:**
   - Mobile-first approach
   - Breakpoints Tailwind (sm, md, lg, xl)
   - Grid adaptatif

---

## 📦 Dépendances

### Bibliothèques NPM

```bash
# Core (déjà installées en Phase 1)
npm install next@14 react react-dom typescript

# UI & Styling (déjà installées en Phase 1)
npm install tailwindcss @tailwindcss/typography
npm install @radix-ui/react-select @radix-ui/react-dropdown-menu
npm install lucide-react # Icons

# Animations
npm install framer-motion

# Forms & Validation
npm install react-hook-form zod @hookform/resolvers

# Utilities
npm install clsx tailwind-merge
npm install date-fns # Date formatting
```

### Services Externes

- **Backend API:** Prérequis 06. BACKEND_CV_API
  - `GET /api/cv?theme={theme}`
  - `GET /api/cv/themes`
  - `GET /api/cv/export?theme={theme}&format=pdf`

---

## 🔨 Implémentation

### Étape 1: Structure de la Page CV

**Fichier:** `frontend/app/cv/page.tsx`

**Description:** Page principale du CV avec gestion des query params et fetch des données

**Code:**

```typescript
import { Suspense } from 'react';
import { Metadata } from 'next';
import CVThemeSelector from '@/components/cv/CVThemeSelector';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import ExportPDFButton from '@/components/cv/ExportPDFButton';
import { CVSkeleton } from '@/components/cv/CVSkeleton';

interface CVPageProps {
  searchParams: {
    theme?: string;
  };
}

// Types
interface CVData {
  theme: string;
  experiences: Experience[];
  skills: Skill[];
  projects: Project[];
}

interface Experience {
  id: string;
  title: string;
  company: string;
  description: string;
  startDate: string;
  endDate?: string;
  technologies: string[];
  tags: string[];
  score?: number;
}

interface Skill {
  id: string;
  name: string;
  level: number;
  category: string;
  yearsExperience: number;
  score?: number;
}

interface Project {
  id: string;
  title: string;
  description: string;
  githubUrl?: string;
  demoUrl?: string;
  technologies: string[];
  stars?: number;
  language?: string;
  featured: boolean;
  score?: number;
}

// Fetch CV data
async function getCVData(theme: string = 'fullstack'): Promise<CVData> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/cv?theme=${theme}`,
    {
      next: { revalidate: 3600 }, // Cache 1 hour
    }
  );

  if (!res.ok) {
    throw new Error('Failed to fetch CV data');
  }

  return res.json();
}

// Generate dynamic metadata
export async function generateMetadata({
  searchParams,
}: CVPageProps): Promise<Metadata> {
  const theme = searchParams.theme || 'fullstack';

  return {
    title: `CV ${theme.charAt(0).toUpperCase() + theme.slice(1)} - Alexi`,
    description: `Découvrez mon profil ${theme} avec mes expériences, compétences et projets pertinents.`,
    openGraph: {
      title: `CV ${theme.charAt(0).toUpperCase() + theme.slice(1)} - Alexi`,
      description: `Profil ${theme} adapté`,
      type: 'profile',
    },
  };
}

// Main CV Page Component
export default async function CVPage({ searchParams }: CVPageProps) {
  const theme = searchParams.theme || 'fullstack';
  const cvData = await getCVData(theme);

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      {/* Header */}
      <header className="mb-12 text-center">
        <h1 className="text-4xl md:text-5xl font-bold mb-4 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          Alexi - CV Dynamique
        </h1>
        <p className="text-lg text-gray-600 dark:text-gray-300 mb-6">
          CV adapté au profil : <span className="font-semibold">{theme}</span>
        </p>

        {/* Theme Selector & Export Button */}
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
          <CVThemeSelector currentTheme={theme} />
          <ExportPDFButton theme={theme} />
        </div>
      </header>

      {/* Main Content */}
      <Suspense fallback={<CVSkeleton />}>
        <main className="space-y-16">
          {/* Experiences Section */}
          <section id="experiences">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-blue-600">💼</span>
              Expériences Professionnelles
            </h2>
            <ExperienceTimeline experiences={cvData.experiences} />
          </section>

          {/* Skills Section */}
          <section id="skills">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-purple-600">🎯</span>
              Compétences
            </h2>
            <SkillsCloud skills={cvData.skills} />
          </section>

          {/* Projects Section */}
          <section id="projects">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-green-600">🚀</span>
              Projets
            </h2>
            <ProjectsGrid projects={cvData.projects} />
          </section>
        </main>
      </Suspense>
    </div>
  );
}
```

**Explications:**
- **Server Component** pour SEO et performance
- **Query params** pour thème dynamique
- **Metadata dynamique** pour partage social
- **Suspense** pour loading states élégants
- **Revalidation** pour cache intelligent

---

### Étape 2: CVThemeSelector Component

**Fichier:** `frontend/components/cv/CVThemeSelector.tsx`

**Description:** Dropdown pour sélectionner le thème du CV avec preview

**Code:**

```typescript
'use client';

import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Sparkles } from 'lucide-react';

interface Theme {
  id: string;
  name: string;
  description: string;
  icon: string;
}

// Fetch available themes from API
async function fetchThemes(): Promise<Theme[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/cv/themes`);
  if (!res.ok) throw new Error('Failed to fetch themes');
  return res.json();
}

interface CVThemeSelectorProps {
  currentTheme: string;
}

export default function CVThemeSelector({ currentTheme }: CVThemeSelectorProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [themes, setThemes] = useState<Theme[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchThemes()
      .then(setThemes)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleThemeChange = (newTheme: string) => {
    const params = new URLSearchParams(searchParams);
    params.set('theme', newTheme);
    router.push(`/cv?${params.toString()}`);
  };

  if (loading) {
    return (
      <div className="w-64 h-10 bg-gray-200 dark:bg-gray-700 rounded-md animate-pulse" />
    );
  }

  return (
    <div className="relative">
      <Select value={currentTheme} onValueChange={handleThemeChange}>
        <SelectTrigger className="w-64 bg-white dark:bg-gray-800 border-2 border-gray-300 dark:border-gray-600 hover:border-blue-500 transition-colors">
          <Sparkles className="w-4 h-4 mr-2 text-blue-600" />
          <SelectValue placeholder="Sélectionner un thème" />
        </SelectTrigger>
        <SelectContent>
          {themes.map((theme) => (
            <SelectItem key={theme.id} value={theme.id}>
              <div className="flex items-center gap-2">
                <span>{theme.icon}</span>
                <div>
                  <div className="font-medium">{theme.name}</div>
                  <div className="text-xs text-gray-500">{theme.description}</div>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
```

**Explications:**
- **Client Component** pour interactivité
- **useRouter** pour navigation programmatique
- **Fetch API** pour récupérer thèmes disponibles
- **shadcn/ui Select** pour dropdown élégant
- **Loading state** avec skeleton

---

### Étape 3: ExperienceTimeline Component

**Fichier:** `frontend/components/cv/ExperienceTimeline.tsx`

**Description:** Timeline verticale des expériences avec animations scroll

**Code:**

```typescript
'use client';

import { motion } from 'framer-motion';
import { format } from 'date-fns';
import { fr } from 'date-fns/locale';
import { Briefcase, Calendar } from 'lucide-react';

interface Experience {
  id: string;
  title: string;
  company: string;
  description: string;
  startDate: string;
  endDate?: string;
  technologies: string[];
  tags: string[];
  score?: number;
}

interface ExperienceTimelineProps {
  experiences: Experience[];
}

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.2,
    },
  },
};

const itemVariants = {
  hidden: { opacity: 0, x: -50 },
  visible: {
    opacity: 1,
    x: 0,
    transition: {
      duration: 0.6,
      ease: 'easeOut',
    },
  },
};

export default function ExperienceTimeline({ experiences }: ExperienceTimelineProps) {
  const formatDate = (dateString: string) => {
    return format(new Date(dateString), 'MMM yyyy', { locale: fr });
  };

  const calculateDuration = (start: string, end?: string) => {
    const startDate = new Date(start);
    const endDate = end ? new Date(end) : new Date();
    const months = Math.round(
      (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24 * 30)
    );
    const years = Math.floor(months / 12);
    const remainingMonths = months % 12;

    if (years > 0 && remainingMonths > 0) {
      return `${years} an${years > 1 ? 's' : ''} ${remainingMonths} mois`;
    } else if (years > 0) {
      return `${years} an${years > 1 ? 's' : ''}`;
    } else {
      return `${months} mois`;
    }
  };

  return (
    <motion.div
      className="relative"
      variants={containerVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-100px' }}
    >
      {/* Vertical Line */}
      <div className="absolute left-8 md:left-1/2 top-0 bottom-0 w-0.5 bg-gradient-to-b from-blue-600 via-purple-600 to-pink-600" />

      {/* Timeline Items */}
      <div className="space-y-12">
        {experiences.map((exp, index) => (
          <motion.div
            key={exp.id}
            variants={itemVariants}
            className={`relative flex items-center ${
              index % 2 === 0 ? 'md:flex-row' : 'md:flex-row-reverse'
            }`}
          >
            {/* Timeline Dot */}
            <div className="absolute left-8 md:left-1/2 w-4 h-4 rounded-full bg-blue-600 border-4 border-white dark:border-gray-900 transform -translate-x-1/2 z-10" />

            {/* Content Card */}
            <div
              className={`ml-20 md:ml-0 md:w-5/12 ${
                index % 2 === 0 ? 'md:mr-auto md:pr-12' : 'md:ml-auto md:pl-12'
              }`}
            >
              <motion.div
                className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 border border-gray-200 dark:border-gray-700 hover:shadow-xl transition-shadow"
                whileHover={{ scale: 1.02 }}
                transition={{ duration: 0.2 }}
              >
                {/* Header */}
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-1">
                      {exp.title}
                    </h3>
                    <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 font-medium">
                      <Briefcase className="w-4 h-4" />
                      <span>{exp.company}</span>
                    </div>
                  </div>
                  {exp.score && (
                    <div className="ml-4 px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full text-sm font-semibold">
                      {Math.round(exp.score * 100)}%
                    </div>
                  )}
                </div>

                {/* Date & Duration */}
                <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 mb-4">
                  <Calendar className="w-4 h-4" />
                  <span>
                    {formatDate(exp.startDate)} - {exp.endDate ? formatDate(exp.endDate) : 'Présent'}
                  </span>
                  <span className="text-gray-400">•</span>
                  <span>{calculateDuration(exp.startDate, exp.endDate)}</span>
                </div>

                {/* Description */}
                <p className="text-gray-700 dark:text-gray-300 mb-4 leading-relaxed">
                  {exp.description}
                </p>

                {/* Technologies */}
                <div className="flex flex-wrap gap-2">
                  {exp.technologies.map((tech) => (
                    <span
                      key={tech}
                      className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full text-sm font-medium"
                    >
                      {tech}
                    </span>
                  ))}
                </div>
              </motion.div>
            </div>
          </motion.div>
        ))}
      </div>
    </motion.div>
  );
}
```

**Explications:**
- **Framer Motion** pour animations scroll
- **Variants** pour cohérence animations
- **whileInView** pour animation au scroll
- **Responsive** : ligne verticale mobile, alternance desktop
- **Date formatting** avec date-fns
- **Score badge** si disponible (pertinence)

---

### Étape 4: SkillsCloud Component

**Fichier:** `frontend/components/cv/SkillsCloud.tsx`

**Description:** Tag cloud interactif avec tailles proportionnelles au niveau

**Code:**

```typescript
'use client';

import { motion } from 'framer-motion';
import { useState } from 'react';

interface Skill {
  id: string;
  name: string;
  level: number; // 1-5
  category: string;
  yearsExperience: number;
  score?: number;
}

interface SkillsCloudProps {
  skills: Skill[];
}

const categoryColors: Record<string, string> = {
  backend: 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 border-blue-300 dark:border-blue-700',
  frontend: 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200 border-purple-300 dark:border-purple-700',
  devops: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 border-green-300 dark:border-green-700',
  database: 'bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 border-orange-300 dark:border-orange-700',
  tools: 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-600',
  other: 'bg-pink-100 dark:bg-pink-900 text-pink-800 dark:text-pink-200 border-pink-300 dark:border-pink-700',
};

export default function SkillsCloud({ skills }: SkillsCloudProps) {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  // Calculate font size based on level and score
  const getFontSize = (skill: Skill) => {
    const baseSize = 14;
    const levelMultiplier = skill.level * 2;
    const scoreMultiplier = skill.score ? skill.score * 4 : 0;
    return baseSize + levelMultiplier + scoreMultiplier;
  };

  // Get categories
  const categories = Array.from(new Set(skills.map((s) => s.category)));

  // Filter skills by category
  const filteredSkills = selectedCategory
    ? skills.filter((s) => s.category === selectedCategory)
    : skills;

  return (
    <div className="space-y-6">
      {/* Category Filter */}
      <div className="flex flex-wrap gap-2 justify-center">
        <button
          onClick={() => setSelectedCategory(null)}
          className={`px-4 py-2 rounded-full font-medium transition-all ${
            selectedCategory === null
              ? 'bg-blue-600 text-white shadow-lg scale-105'
              : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
          }`}
        >
          Toutes
        </button>
        {categories.map((category) => (
          <button
            key={category}
            onClick={() => setSelectedCategory(category)}
            className={`px-4 py-2 rounded-full font-medium transition-all capitalize ${
              selectedCategory === category
                ? 'bg-blue-600 text-white shadow-lg scale-105'
                : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
            }`}
          >
            {category}
          </button>
        ))}
      </div>

      {/* Skills Cloud */}
      <motion.div
        className="flex flex-wrap gap-3 justify-center items-center p-8 bg-gray-50 dark:bg-gray-900 rounded-lg min-h-[300px]"
        layout
      >
        {filteredSkills.map((skill) => (
          <motion.div
            key={skill.id}
            layout
            initial={{ opacity: 0, scale: 0 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0 }}
            whileHover={{ scale: 1.15, rotate: 2 }}
            transition={{ duration: 0.3 }}
            className={`px-4 py-2 rounded-full font-semibold border-2 cursor-pointer ${
              categoryColors[skill.category] || categoryColors.other
            }`}
            style={{
              fontSize: `${getFontSize(skill)}px`,
            }}
            title={`${skill.name} - Niveau ${skill.level}/5 - ${skill.yearsExperience} ans d'expérience`}
          >
            {skill.name}
          </motion.div>
        ))}
      </motion.div>

      {/* Legend */}
      <div className="text-center text-sm text-gray-600 dark:text-gray-400">
        <p>La taille représente le niveau de compétence et la pertinence par rapport au thème sélectionné</p>
      </div>
    </div>
  );
}
```

**Explications:**
- **Tag cloud interactif** avec tailles dynamiques
- **Filtrage par catégorie** avec animation
- **Hover effects** avec rotation
- **Layout animations** avec Framer Motion
- **Color coding** par catégorie
- **Tooltip** avec niveau et années d'expérience

---

### Étape 5: ProjectsGrid Component

**Fichier:** `frontend/components/cv/ProjectsGrid.tsx`

**Description:** Grid responsive de cards projets avec informations GitHub

**Code:**

```typescript
'use client';

import { motion } from 'framer-motion';
import { ExternalLink, Github, Star } from 'lucide-react';
import Link from 'next/link';

interface Project {
  id: string;
  title: string;
  description: string;
  githubUrl?: string;
  demoUrl?: string;
  technologies: string[];
  stars?: number;
  language?: string;
  featured: boolean;
  score?: number;
}

interface ProjectsGridProps {
  projects: Project[];
}

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.5,
    },
  },
};

const languageColors: Record<string, string> = {
  TypeScript: 'bg-blue-500',
  JavaScript: 'bg-yellow-500',
  Go: 'bg-cyan-500',
  Python: 'bg-green-500',
  Rust: 'bg-orange-500',
  Java: 'bg-red-500',
  'C++': 'bg-purple-500',
};

export default function ProjectsGrid({ projects }: ProjectsGridProps) {
  return (
    <motion.div
      className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
      variants={containerVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true }}
    >
      {projects.map((project) => (
        <motion.div
          key={project.id}
          variants={itemVariants}
          whileHover={{ y: -8 }}
          className={`bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 border-2 transition-all ${
            project.featured
              ? 'border-yellow-400 dark:border-yellow-600'
              : 'border-gray-200 dark:border-gray-700'
          }`}
        >
          {/* Featured Badge */}
          {project.featured && (
            <div className="flex items-center gap-1 text-yellow-600 dark:text-yellow-400 mb-2 font-semibold text-sm">
              <Star className="w-4 h-4 fill-current" />
              <span>Projet Vedette</span>
            </div>
          )}

          {/* Header */}
          <div className="mb-3">
            <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
              {project.title}
            </h3>

            {/* Language & Stars */}
            <div className="flex items-center gap-3 text-sm">
              {project.language && (
                <div className="flex items-center gap-1">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      languageColors[project.language] || 'bg-gray-500'
                    }`}
                  />
                  <span className="text-gray-600 dark:text-gray-400">
                    {project.language}
                  </span>
                </div>
              )}
              {project.stars !== undefined && (
                <div className="flex items-center gap-1 text-gray-600 dark:text-gray-400">
                  <Star className="w-4 h-4" />
                  <span>{project.stars}</span>
                </div>
              )}
            </div>
          </div>

          {/* Description */}
          <p className="text-gray-700 dark:text-gray-300 mb-4 line-clamp-3">
            {project.description}
          </p>

          {/* Technologies */}
          <div className="flex flex-wrap gap-2 mb-4">
            {project.technologies.slice(0, 4).map((tech) => (
              <span
                key={tech}
                className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded text-xs font-medium"
              >
                {tech}
              </span>
            ))}
            {project.technologies.length > 4 && (
              <span className="px-2 py-1 text-gray-500 dark:text-gray-400 text-xs">
                +{project.technologies.length - 4}
              </span>
            )}
          </div>

          {/* Links */}
          <div className="flex gap-3">
            {project.githubUrl && (
              <Link
                href={project.githubUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
              >
                <Github className="w-4 h-4" />
                <span className="text-sm font-medium">Code</span>
              </Link>
            )}
            {project.demoUrl && (
              <Link
                href={project.demoUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
              >
                <ExternalLink className="w-4 h-4" />
                <span className="text-sm font-medium">Demo</span>
              </Link>
            )}
          </div>

          {/* Score Badge */}
          {project.score && (
            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between text-sm">
                <span className="text-gray-600 dark:text-gray-400">Pertinence</span>
                <div className="flex items-center gap-2">
                  <div className="w-24 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-blue-600 rounded-full transition-all"
                      style={{ width: `${project.score * 100}%` }}
                    />
                  </div>
                  <span className="font-semibold text-blue-600">
                    {Math.round(project.score * 100)}%
                  </span>
                </div>
              </div>
            </div>
          )}
        </motion.div>
      ))}
    </motion.div>
  );
}
```

**Explications:**
- **Grid responsive** avec Tailwind
- **Featured badge** pour projets vedettes
- **GitHub stats** (stars, language)
- **Hover lift effect** avec Framer Motion
- **Stagger animations** pour apparition progressive
- **Progress bar** pour score de pertinence
- **Line clamp** pour description tronquée

---

### Étape 6: ExportPDFButton Component

**Fichier:** `frontend/components/cv/ExportPDFButton.tsx`

**Description:** Bouton pour exporter le CV en PDF

**Code:**

```typescript
'use client';

import { useState } from 'react';
import { Download, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface ExportPDFButtonProps {
  theme: string;
}

export default function ExportPDFButton({ theme }: ExportPDFButtonProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/cv/export?theme=${theme}&format=pdf`
      );

      if (!response.ok) {
        throw new Error('Échec de l\'export PDF');
      }

      // Get filename from Content-Disposition header
      const contentDisposition = response.headers.get('Content-Disposition');
      const filenameMatch = contentDisposition?.match(/filename="(.+)"/);
      const filename = filenameMatch ? filenameMatch[1] : `CV_${theme}.pdf`;

      // Download file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Une erreur est survenue');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center gap-2">
      <Button
        onClick={handleExport}
        disabled={loading}
        className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white font-semibold px-6 py-2 rounded-lg shadow-lg hover:shadow-xl transition-all"
      >
        {loading ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            Export en cours...
          </>
        ) : (
          <>
            <Download className="w-4 h-4 mr-2" />
            Télécharger PDF
          </>
        )}
      </Button>
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  );
}
```

**Explications:**
- **Client Component** pour interaction
- **Loading state** avec spinner
- **Error handling** avec message
- **Blob download** pour fichier PDF
- **Filename extraction** depuis headers
- **shadcn/ui Button** pour style cohérent

---

### Étape 7: CVSkeleton Component (Loading State)

**Fichier:** `frontend/components/cv/CVSkeleton.tsx`

**Description:** Skeleton loading state pendant fetch des données

**Code:**

```typescript
export function CVSkeleton() {
  return (
    <div className="space-y-16 animate-pulse">
      {/* Experiences Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-64 mb-8" />
        <div className="space-y-8">
          {[1, 2, 3].map((i) => (
            <div key={i} className="ml-20 bg-gray-200 dark:bg-gray-700 rounded-lg h-48" />
          ))}
        </div>
      </section>

      {/* Skills Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-48 mb-8" />
        <div className="flex flex-wrap gap-3 p-8">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
            <div
              key={i}
              className="h-10 bg-gray-200 dark:bg-gray-700 rounded-full"
              style={{ width: `${80 + Math.random() * 80}px` }}
            />
          ))}
        </div>
      </section>

      {/* Projects Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-56 mb-8" />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="bg-gray-200 dark:bg-gray-700 rounded-lg h-64" />
          ))}
        </div>
      </section>
    </div>
  );
}
```

**Explications:**
- **Skeleton screens** pour UX fluide
- **animate-pulse** Tailwind pour effet shimmer
- **Même layout** que contenu réel
- **Hauteurs approximatives** pour éviter layout shift

---

## 🧪 Tests

### Tests Unitaires (Jest + React Testing Library)

**Fichier:** `frontend/components/cv/__tests__/CVThemeSelector.test.tsx`

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { useRouter, useSearchParams } from 'next/navigation';
import CVThemeSelector from '../CVThemeSelector';

// Mock Next.js navigation
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
  useSearchParams: jest.fn(),
}));

// Mock fetch
global.fetch = jest.fn();

describe('CVThemeSelector', () => {
  const mockPush = jest.fn();
  const mockSearchParams = new URLSearchParams();

  beforeEach(() => {
    (useRouter as jest.Mock).mockReturnValue({ push: mockPush });
    (useSearchParams as jest.Mock).mockReturnValue(mockSearchParams);
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => [
        { id: 'backend', name: 'Backend', description: 'Backend dev', icon: '⚙️' },
        { id: 'frontend', name: 'Frontend', description: 'Frontend dev', icon: '🎨' },
      ],
    });
  });

  it('renders theme selector', async () => {
    render(<CVThemeSelector currentTheme="backend" />);
    await waitFor(() => {
      expect(screen.getByText('Backend')).toBeInTheDocument();
    });
  });

  it('changes theme on selection', async () => {
    render(<CVThemeSelector currentTheme="backend" />);

    await waitFor(() => {
      expect(screen.getByText('Backend')).toBeInTheDocument();
    });

    // Open dropdown and select frontend
    const trigger = screen.getByRole('combobox');
    fireEvent.click(trigger);

    const frontendOption = screen.getByText('Frontend');
    fireEvent.click(frontendOption);

    expect(mockPush).toHaveBeenCalledWith('/cv?theme=frontend');
  });
});
```

### Tests E2E (Playwright)

**Fichier:** `e2e/cv.spec.ts`

```typescript
import { test, expect } from '@playwright/test';

test.describe('CV Dynamic Page', () => {
  test('should load CV with default theme', async ({ page }) => {
    await page.goto('/cv');

    // Check main elements
    await expect(page.locator('h1')).toContainText('CV Dynamique');
    await expect(page.locator('#experiences')).toBeVisible();
    await expect(page.locator('#skills')).toBeVisible();
    await expect(page.locator('#projects')).toBeVisible();
  });

  test('should change theme via selector', async ({ page }) => {
    await page.goto('/cv?theme=backend');

    // Check theme is applied
    await expect(page).toHaveURL(/theme=backend/);

    // Change theme
    await page.click('[role="combobox"]');
    await page.click('text=Frontend');

    // Verify URL updated
    await expect(page).toHaveURL(/theme=frontend/);
  });

  test('should export PDF', async ({ page }) => {
    await page.goto('/cv?theme=backend');

    // Click export button
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.click('text=Télécharger PDF'),
    ]);

    // Verify download
    expect(download.suggestedFilename()).toContain('.pdf');
  });

  test('should animate on scroll', async ({ page }) => {
    await page.goto('/cv');

    // Get initial position of experience timeline
    const timeline = page.locator('#experiences');
    await expect(timeline).toBeVisible();

    // Scroll to skills section
    await page.locator('#skills').scrollIntoViewIfNeeded();

    // Check animation classes (Framer Motion adds classes)
    const skills = page.locator('#skills');
    await expect(skills).toBeVisible();
  });
});
```

### Commandes

```bash
# Run unit tests
npm test

# Run e2e tests
npm run test:e2e

# Run with coverage
npm test -- --coverage

# Watch mode
npm test -- --watch
```

---

## ⚠️ Points d'Attention

### Pièges à Éviter

- **Layout Shift:** Utiliser skeleton avec dimensions fixes pour éviter layout shift pendant loading
- **Animations Lourdes:** Limiter animations complexes sur mobile (performances)
- **Fetch Waterfalls:** Utiliser Server Components pour fetch côté serveur et éviter waterfalls
- **Cache Stale:** Configurer revalidation appropriée (1h pour CV data)

### Edge Cases

- **Thème Invalide:** Gérer query param invalide → fallback sur "fullstack"
- **API Indisponible:** Error boundary pour afficher message gracieux
- **Pas de Données:** Gérer cas où experiences/skills/projects sont vides
- **Export PDF Timeout:** Timeout de 30s pour génération PDF, afficher erreur si dépassé

### Optimisations

- **Image Optimization:** Utiliser `next/image` pour logos entreprises/projets
- **Code Splitting:** Lazy load components lourds (ProjectsGrid si nombreux projets)
- **Prefetching:** Prefetch autres thèmes au hover du selector
- **Memoization:** Usememo pour calculs coûteux (fontSize, filtering)

### Accessibilité

- **Keyboard Navigation:** Tous les interactifs accessibles au clavier
- **ARIA Labels:** Labels explicites pour lecteurs d'écran
- **Focus Visible:** Outline visible sur focus (pas `outline-none`)
- **Color Contrast:** Ratio 4.5:1 minimum (WCAG AA)

---

## 📚 Ressources

### Documentation Officielle

- [Next.js App Router](https://nextjs.org/docs/app)
- [Framer Motion](https://www.framer.com/motion/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [shadcn/ui](https://ui.shadcn.com/)

### Tutoriels

- [Framer Motion Scroll Animations](https://www.framer.com/motion/scroll-animations/)
- [Next.js Data Fetching](https://nextjs.org/docs/app/building-your-application/data-fetching)
- [TypeScript Best Practices](https://www.typescriptlang.org/docs/handbook/declaration-files/do-s-and-don-ts.html)

### Inspiration Design

- [Dribbble CV Designs](https://dribbble.com/search/cv-design)
- [Awwwards Portfolio Sites](https://www.awwwards.com/websites/portfolio/)

---

## ✅ Checklist de Complétion

- [ ] Page /cv avec routing query params
- [ ] CVThemeSelector component avec fetch themes
- [ ] ExperienceTimeline avec animations scroll
- [ ] SkillsCloud avec filtrage et hover effects
- [ ] ProjectsGrid responsive avec GitHub stats
- [ ] ExportPDFButton avec download
- [ ] CVSkeleton loading state
- [ ] Tests unitaires (components isolés)
- [ ] Tests e2e (scénarios utilisateur)
- [ ] Responsive design (mobile, tablet, desktop)
- [ ] Animations Framer Motion fluides
- [ ] SEO metadata dynamique
- [ ] Error handling et error boundaries
- [ ] Accessibilité (keyboard, ARIA, contrast)
- [ ] Performance optimization (lazy load, memoization)
- [ ] Documentation code (commentaires)
- [ ] Review sécurité (XSS prevention)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
