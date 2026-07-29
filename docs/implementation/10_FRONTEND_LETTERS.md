# 10. FRONTEND LETTERS - Interface de Génération de Lettres par IA

## 📋 Métadonnées

- **Phase:** 3
- **Priorité:** HAUTE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 05. FRONTEND_FOUNDATION.md, 09. BACKEND_LETTERS_API.md
- **Temps estimé:** 4-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Créer l'interface utilisateur complète pour la génération de lettres de motivation et anti-motivation par IA, avec affichage dual (2 lettres côte à côte), système d'access gate (teaser avant 3 visites), gestion des états de chargement, et export PDF.

**Particularité clé:** L'affichage dual permet de visualiser simultanément la lettre de motivation professionnelle et la lettre d'anti-motivation humoristique, offrant une expérience unique et mémorable.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
/letters (Page)
├── AccessGate (vérification 3 visites)
│   ├── Teaser (si bloqué)
│   └── LetterGenerator (si autorisé)
│       ├── Form (input entreprise)
│       ├── Loading State (génération en cours)
│       └── LetterPreview (dual display)
│           ├── MotivationLetter (gauche)
│           ├── AntiMotivationLetter (droite)
│           └── Export Controls (PDF)
```

### Flux Utilisateur

```
1. Arrivée sur /letters
   ↓
2. Vérification compteur visites (API)
   ↓
3a. Si < 3 visites → Teaser + compteur
3b. Si ≥ 3 visites OU profil détecté → Formulaire accessible
   ↓
4. Saisie nom entreprise + Submit
   ↓
5. Loading state (animation IA)
   ↓
6. Affichage dual des 2 lettres
   ↓
7. Options export PDF (individuel ou dual)
```

### Design Decisions

**1. Affichage Dual (2 colonnes):**
- Desktop: 2 colonnes côte à côte (50/50)
- Tablet: 2 colonnes scrollables
- Mobile: Stack vertical (motivation en haut, anti-motivation en bas)

**2. Access Gate Strategy:**
- Vérification côté client (API `/api/visitors/check`)
- Fallback serveur (403 Forbidden si contournement)
- Message encourageant pour inciter aux revisites

**3. State Management:**
- React hooks (useState, useEffect)
- Pas besoin de Redux/Zustand (scope limité à cette page)
- Cache local des lettres générées (localStorage)

**4. Error Handling:**
- Messages contextuels selon code erreur (403, 429, 500)
- Retry automatique avec exponential backoff
- Offline detection

---

## 📦 Dépendances

### Bibliothèques NPM

```bash
# Validation forms
npm install react-hook-form zod @hookform/resolvers

# Markdown rendering
npm install react-markdown remark-gfm

# PDF export
npm install jspdf html2canvas

# Animations
npm install framer-motion

# UI components (si pas déjà installés)
npm install @radix-ui/react-dialog @radix-ui/react-progress

# Syntax highlighting pour code blocks
npm install react-syntax-highlighter @types/react-syntax-highlighter
```

### Types TypeScript

```bash
npm install -D @types/react-syntax-highlighter
```

---

## 🔨 Implémentation

### Étape 1: Page Letters (/letters/page.tsx)

**Description:** Page principale qui orchestre tous les composants

**Code:**

```tsx
// frontend/app/letters/page.tsx
import { Metadata } from 'next';
import { LetterGenerator } from '@/components/letters/LetterGenerator';
import { AccessGate } from '@/components/letters/AccessGate';

export const metadata: Metadata = {
  title: 'Générateur de Lettres IA | maicivy',
  description: 'Générez des lettres de motivation et anti-motivation personnalisées par IA pour vos candidatures.',
  openGraph: {
    title: 'Générateur de Lettres IA',
    description: 'Lettres de motivation/anti-motivation générées par IA',
    images: ['/og-letters.png'],
  },
};

export default function LettersPage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-blue-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
      <div className="container mx-auto px-4 py-12">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-bold mb-4 bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-purple-600 dark:from-blue-400 dark:to-purple-400">
            Générateur de Lettres par IA
          </h1>
          <p className="text-lg text-slate-600 dark:text-slate-300 max-w-2xl mx-auto">
            Générez instantanément une lettre de motivation professionnelle et sa version humoristique "anti-motivation"
          </p>
        </div>

        {/* Access Gate + Generator */}
        <AccessGate>
          <LetterGenerator />
        </AccessGate>
      </div>
    </div>
  );
}
```

**Explications:**
- Metadata pour SEO
- Layout responsive avec gradient background
- AccessGate wrapper pour vérification d'accès
- LetterGenerator comme enfant (rendu conditionnel)

---

### Étape 2: AccessGate Component

**Description:** Vérifie le nombre de visites et affiche teaser ou contenu

**Code:**

```tsx
// frontend/components/letters/AccessGate.tsx
'use client';

import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Lock, Eye, Sparkles } from 'lucide-react';
import { apiClient } from '@/lib/api';

interface AccessGateProps {
  children: React.ReactNode;
}

interface VisitorStatus {
  visitCount: number;
  hasAccess: boolean;
  profileDetected?: string;
  remainingVisits: number;
}

export function AccessGate({ children }: AccessGateProps) {
  const [status, setStatus] = useState<VisitorStatus | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkAccess();
  }, []);

  const checkAccess = async () => {
    try {
      const response = await apiClient.get<VisitorStatus>('/api/visitors/check');
      setStatus(response.data);
    } catch (error) {
      console.error('Error checking access:', error);
      // Fallback: permettre l'accès en cas d'erreur (serveur vérifiera)
      setStatus({
        visitCount: 0,
        hasAccess: true,
        remainingVisits: 0,
      });
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  // Si accès autorisé
  if (status?.hasAccess) {
    return <>{children}</>;
  }

  // Teaser (accès refusé)
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="max-w-2xl mx-auto"
    >
      <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">
        {/* Icon */}
        <div className="flex justify-center mb-6">
          <div className="relative">
            <div className="absolute inset-0 bg-blue-500/20 blur-xl rounded-full"></div>
            <div className="relative bg-gradient-to-br from-blue-500 to-purple-500 p-4 rounded-full">
              <Lock className="w-8 h-8 text-white" />
            </div>
          </div>
        </div>

        {/* Title */}
        <h2 className="text-2xl font-bold text-center mb-4 text-slate-900 dark:text-white">
          Fonctionnalité Premium
        </h2>

        {/* Description */}
        <p className="text-center text-slate-600 dark:text-slate-300 mb-6">
          Le générateur de lettres par IA est accessible à partir de la{' '}
          <span className="font-bold text-blue-600 dark:text-blue-400">3ème visite</span>.
        </p>

        {/* Progress */}
        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
              Votre progression
            </span>
            <span className="text-sm font-bold text-blue-600 dark:text-blue-400">
              {status?.visitCount || 0} / 3 visites
            </span>
          </div>
          <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-3 overflow-hidden">
            <motion.div
              initial={{ width: 0 }}
              animate={{ width: `${((status?.visitCount || 0) / 3) * 100}%` }}
              transition={{ duration: 1, ease: 'easeOut' }}
              className="bg-gradient-to-r from-blue-500 to-purple-500 h-full rounded-full"
            />
          </div>
        </div>

        {/* Remaining visits */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-6">
          <div className="flex items-start gap-3">
            <Eye className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                Encore {status?.remainingVisits || 3} visite{status?.remainingVisits !== 1 ? 's' : ''} avant déblocage
              </p>
              <p className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                Revenez explorer mon CV pour débloquer cette fonctionnalité !
              </p>
            </div>
          </div>
        </div>

        {/* Features preview */}
        <div className="space-y-3">
          <p className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
            Vous débloquerez :
          </p>
          {[
            'Génération de lettre de motivation personnalisée',
            'Lettre d\'anti-motivation humoristique unique',
            'Export PDF professionnel des deux lettres',
            'Analyse IA de l\'entreprise cible',
          ].map((feature, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.1 }}
              className="flex items-center gap-2"
            >
              <Sparkles className="w-4 h-4 text-purple-500 flex-shrink-0" />
              <span className="text-sm text-slate-600 dark:text-slate-400">{feature}</span>
            </motion.div>
          ))}
        </div>

        {/* CTA */}
        <div className="mt-8 text-center">
          <a
            href="/cv"
            className="inline-flex items-center gap-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white px-6 py-3 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl"
          >
            Explorer mon CV
          </a>
        </div>
      </div>
    </motion.div>
  );
}
```

**Explications:**
- Appel API au montage pour vérifier le statut visiteur
- Affichage conditionnel: teaser ou children
- Barre de progression animée (Framer Motion)
- Design engageant pour inciter aux revisites
- CTA vers /cv pour encourager l'exploration

---

### Étape 3: LetterGenerator Component

**Description:** Formulaire de génération + orchestration des états

**Code:**

```tsx
// frontend/components/letters/LetterGenerator.tsx
'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { motion } from 'framer-motion';
import { Loader2, Sparkles } from 'lucide-react';
import { apiClient } from '@/lib/api';
import { LetterPreview } from './LetterPreview';

// Validation schema
const formSchema = z.object({
  companyName: z.string()
    .min(2, 'Le nom doit contenir au moins 2 caractères')
    .max(100, 'Le nom ne peut pas dépasser 100 caractères')
    .regex(/^[a-zA-Z0-9\s\-&.,'À-ÿ]+$/, 'Caractères invalides détectés'),
});

type FormData = z.infer<typeof formSchema>;

interface GeneratedLetters {
  id: string;
  companyName: string;
  motivationLetter: string;
  antiMotivationLetter: string;
  companyInfo?: {
    industry?: string;
    description?: string;
    website?: string;
  };
  createdAt: string;
}

export function LetterGenerator() {
  const [letters, setLetters] = useState<GeneratedLetters | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<FormData>({
    resolver: zodResolver(formSchema),
  });

  const onSubmit = async (data: FormData) => {
    setIsGenerating(true);
    setError(null);
    setProgress(0);

    // Simulation de progression (réel: utiliser WebSocket ou polling)
    const progressInterval = setInterval(() => {
      setProgress((prev) => Math.min(prev + 10, 90));
    }, 800);

    try {
      const response = await apiClient.post<GeneratedLetters>(
        '/api/letters/generate',
        { companyName: data.companyName }
      );

      setLetters(response.data);
      setProgress(100);

      // Sauvegarder dans localStorage (historique)
      saveToHistory(response.data);
    } catch (err: any) {
      clearInterval(progressInterval);
      handleError(err);
    } finally {
      clearInterval(progressInterval);
      setIsGenerating(false);
    }
  };

  const handleError = (err: any) => {
    const status = err.response?.status;
    const message = err.response?.data?.error || err.message;

    switch (status) {
      case 403:
        setError('Accès refusé. Vous devez effectuer 3 visites pour débloquer cette fonctionnalité.');
        break;
      case 429:
        const retryAfter = err.response?.headers['retry-after'] || 120;
        setError(`Limite atteinte. Réessayez dans ${Math.ceil(retryAfter / 60)} minutes.`);
        break;
      case 500:
        setError('Erreur serveur. Nos IA prennent une pause café. Réessayez dans quelques instants.');
        break;
      default:
        setError(message || 'Une erreur est survenue lors de la génération.');
    }
  };

  const saveToHistory = (data: GeneratedLetters) => {
    try {
      const history = JSON.parse(localStorage.getItem('letters_history') || '[]');
      history.unshift({
        id: data.id,
        companyName: data.companyName,
        createdAt: data.createdAt,
      });
      // Garder seulement les 10 dernières
      localStorage.setItem('letters_history', JSON.stringify(history.slice(0, 10)));
    } catch (e) {
      console.error('Failed to save to history:', e);
    }
  };

  const handleReset = () => {
    setLetters(null);
    setError(null);
    setProgress(0);
    reset();
  };

  return (
    <div className="max-w-7xl mx-auto">
      {/* Form */}
      {!letters && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="max-w-2xl mx-auto"
        >
          <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
              {/* Input */}
              <div>
                <label
                  htmlFor="companyName"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  Nom de l'entreprise
                </label>
                <input
                  {...register('companyName')}
                  id="companyName"
                  type="text"
                  placeholder="Ex: Google, Microsoft, Startup Innovante..."
                  disabled={isGenerating}
                  className="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                />
                {errors.companyName && (
                  <p className="mt-2 text-sm text-red-600 dark:text-red-400">
                    {errors.companyName.message}
                  </p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isGenerating}
                className="w-full bg-gradient-to-r from-blue-600 to-purple-600 text-white px-6 py-4 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                {isGenerating ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Génération en cours...
                  </>
                ) : (
                  <>
                    <Sparkles className="w-5 h-5" />
                    Générer les lettres
                  </>
                )}
              </button>

              {/* Progress Bar */}
              {isGenerating && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-600 dark:text-slate-400">
                      {progress < 30 && 'Analyse de l\'entreprise...'}
                      {progress >= 30 && progress < 60 && 'Rédaction de la lettre de motivation...'}
                      {progress >= 60 && progress < 90 && 'Création de l\'anti-motivation...'}
                      {progress >= 90 && 'Finalisation...'}
                    </span>
                    <span className="font-medium text-blue-600 dark:text-blue-400">
                      {progress}%
                    </span>
                  </div>
                  <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2 overflow-hidden">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${progress}%` }}
                      className="bg-gradient-to-r from-blue-500 to-purple-500 h-full"
                    />
                  </div>
                </div>
              )}

              {/* Error Display */}
              {error && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"
                >
                  <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </motion.div>
              )}

              {/* Info */}
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <p className="text-xs text-blue-800 dark:text-blue-200">
                  L'IA va générer deux lettres : une motivation professionnelle et une anti-motivation humoristique.
                  La génération prend environ 30-60 secondes.
                </p>
              </div>
            </form>
          </div>
        </motion.div>
      )}

      {/* Preview */}
      {letters && (
        <LetterPreview letters={letters} onReset={handleReset} />
      )}
    </div>
  );
}
```

**Explications:**
- React Hook Form + Zod pour validation robuste
- Gestion états: loading, error, success
- Simulation progression (à remplacer par WebSocket/polling en prod)
- Sauvegarde historique dans localStorage
- Error handling contextuel selon code HTTP
- Design responsive et accessible

---

### Étape 4: LetterPreview Component (Dual Display)

**Description:** Affichage des 2 lettres côte à côte + export PDF

**Code:**

```tsx
// frontend/components/letters/LetterPreview.tsx
'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Download, RotateCcw, Copy, Check } from 'lucide-react';
import { apiClient } from '@/lib/api';

interface LetterPreviewProps {
  letters: {
    id: string;
    companyName: string;
    motivationLetter: string;
    antiMotivationLetter: string;
    companyInfo?: {
      industry?: string;
      description?: string;
      website?: string;
    };
  };
  onReset: () => void;
}

export function LetterPreview({ letters, onReset }: LetterPreviewProps) {
  const [downloadingMotivation, setDownloadingMotivation] = useState(false);
  const [downloadingAnti, setDownloadingAnti] = useState(false);
  const [downloadingBoth, setDownloadingBoth] = useState(false);
  const [copiedMotivation, setCopiedMotivation] = useState(false);
  const [copiedAnti, setCopiedAnti] = useState(false);

  const downloadPDF = async (type: 'motivation' | 'anti' | 'both') => {
    const setLoading = {
      motivation: setDownloadingMotivation,
      anti: setDownloadingAnti,
      both: setDownloadingBoth,
    }[type];

    setLoading(true);

    try {
      const response = await apiClient.get(`/api/letters/${letters.id}/pdf`, {
        params: { type },
        responseType: 'blob',
      });

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `lettre-${letters.companyName}-${type}.pdf`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('PDF download failed:', error);
      alert('Erreur lors du téléchargement du PDF');
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = async (text: string, type: 'motivation' | 'anti') => {
    try {
      await navigator.clipboard.writeText(text);
      if (type === 'motivation') {
        setCopiedMotivation(true);
        setTimeout(() => setCopiedMotivation(false), 2000);
      } else {
        setCopiedAnti(true);
        setTimeout(() => setCopiedAnti(false), 2000);
      }
    } catch (error) {
      console.error('Copy failed:', error);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="space-y-6"
    >
      {/* Header avec actions */}
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6 border border-slate-200 dark:border-slate-700">
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
          <div>
            <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-1">
              Lettres pour {letters.companyName}
            </h2>
            {letters.companyInfo?.industry && (
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Secteur: {letters.companyInfo.industry}
              </p>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => downloadPDF('both')}
              disabled={downloadingBoth}
              className="inline-flex items-center gap-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white px-4 py-2 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all disabled:opacity-50 text-sm"
            >
              {downloadingBoth ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <Download className="w-4 h-4" />
              )}
              PDF Dual
            </button>

            <button
              onClick={onReset}
              className="inline-flex items-center gap-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 px-4 py-2 rounded-lg font-medium hover:bg-slate-300 dark:hover:bg-slate-600 transition-all text-sm"
            >
              <RotateCcw className="w-4 h-4" />
              Nouvelle génération
            </button>
          </div>
        </div>
      </div>

      {/* Dual Display */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Lettre de Motivation */}
        <motion.div
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.1 }}
          className="bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 overflow-hidden"
        >
          {/* Header */}
          <div className="bg-gradient-to-r from-green-500 to-emerald-500 p-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <span className="text-2xl">✅</span>
                Lettre de Motivation
              </h3>
              <div className="flex gap-2">
                <button
                  onClick={() => copyToClipboard(letters.motivationLetter, 'motivation')}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors"
                  title="Copier le texte"
                >
                  {copiedMotivation ? (
                    <Check className="w-4 h-4 text-white" />
                  ) : (
                    <Copy className="w-4 h-4 text-white" />
                  )}
                </button>
                <button
                  onClick={() => downloadPDF('motivation')}
                  disabled={downloadingMotivation}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors disabled:opacity-50"
                  title="Télécharger PDF"
                >
                  {downloadingMotivation ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <Download className="w-4 h-4 text-white" />
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 max-h-[600px] overflow-y-auto">
            <div className="prose prose-slate dark:prose-invert max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {letters.motivationLetter}
              </ReactMarkdown>
            </div>
          </div>
        </motion.div>

        {/* Lettre d'Anti-Motivation */}
        <motion.div
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.2 }}
          className="bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 overflow-hidden"
        >
          {/* Header */}
          <div className="bg-gradient-to-r from-orange-500 to-red-500 p-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <span className="text-2xl">❌</span>
                Lettre d'Anti-Motivation
              </h3>
              <div className="flex gap-2">
                <button
                  onClick={() => copyToClipboard(letters.antiMotivationLetter, 'anti')}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors"
                  title="Copier le texte"
                >
                  {copiedAnti ? (
                    <Check className="w-4 h-4 text-white" />
                  ) : (
                    <Copy className="w-4 h-4 text-white" />
                  )}
                </button>
                <button
                  onClick={() => downloadPDF('anti')}
                  disabled={downloadingAnti}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors disabled:opacity-50"
                  title="Télécharger PDF"
                >
                  {downloadingAnti ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <Download className="w-4 h-4 text-white" />
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 max-h-[600px] overflow-y-auto">
            <div className="prose prose-slate dark:prose-invert max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {letters.antiMotivationLetter}
              </ReactMarkdown>
            </div>
          </div>
        </motion.div>
      </div>

      {/* Info Footer */}
      <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
        <p className="text-sm text-amber-800 dark:text-amber-200">
          <strong>Note:</strong> La lettre d'anti-motivation est générée à titre humoristique et créatif.
          Elle ne doit PAS être envoyée à l'entreprise. Utilisez uniquement la lettre de motivation professionnelle
          pour vos candidatures réelles.
        </p>
      </div>
    </motion.div>
  );
}
```

**Explications:**
- Layout dual: 2 colonnes (grid responsive)
- Distinction visuelle claire: vert (motivation) vs rouge/orange (anti)
- ReactMarkdown pour rendu formaté
- Actions: copy to clipboard, download PDF individuel ou dual
- Scroll indépendant pour chaque lettre (max-height + overflow)
- Avertissement pour anti-motivation

---

### Étape 5: API Client Utilities

**Description:** Helper pour appels API avec error handling

**Code:**

```ts
// frontend/lib/api.ts (ajout/complément)

import axios, { AxiosInstance, AxiosError } from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

class APIClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 60000, // 60s pour génération IA
      withCredentials: true, // Important pour cookies session
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Interceptor pour retry automatique
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const config = error.config as any;

        // Retry logic pour erreurs réseau (max 3 retries)
        if (!config || !config.retry) {
          config.retry = 0;
        }

        if (config.retry < 3 && this.isRetryableError(error)) {
          config.retry += 1;
          const delay = Math.pow(2, config.retry) * 1000; // Exponential backoff
          await new Promise((resolve) => setTimeout(resolve, delay));
          return this.client(config);
        }

        return Promise.reject(error);
      }
    );
  }

  private isRetryableError(error: AxiosError): boolean {
    if (!error.response) return true; // Network error
    const status = error.response.status;
    return status >= 500 || status === 429; // Server errors ou rate limit
  }

  async get<T>(url: string, config?: any) {
    return this.client.get<T>(url, config);
  }

  async post<T>(url: string, data?: any, config?: any) {
    return this.client.post<T>(url, data, config);
  }

  async put<T>(url: string, data?: any, config?: any) {
    return this.client.put<T>(url, data, config);
  }

  async delete<T>(url: string, config?: any) {
    return this.client.delete<T>(url, config);
  }
}

export const apiClient = new APIClient();
```

**Explications:**
- Singleton API client
- `withCredentials: true` pour envoyer cookies (tracking session)
- Timeout long (60s) pour génération IA
- Retry automatique avec exponential backoff
- Gestion erreurs réseau vs erreurs serveur

---

### Étape 6: Types TypeScript

**Description:** Définition des types pour type-safety

**Code:**

```ts
// frontend/types/letters.ts

export interface CompanyInfo {
  industry?: string;
  description?: string;
  website?: string;
  size?: string;
  location?: string;
}

export interface GeneratedLetters {
  id: string;
  companyName: string;
  motivationLetter: string;
  antiMotivationLetter: string;
  companyInfo?: CompanyInfo;
  createdAt: string;
  updatedAt?: string;
}

export interface LetterHistoryItem {
  id: string;
  companyName: string;
  createdAt: string;
}

export interface GenerateLetterRequest {
  companyName: string;
}

export interface GenerateLetterResponse extends GeneratedLetters {}

export interface VisitorStatus {
  visitCount: number;
  hasAccess: boolean;
  profileDetected?: string;
  remainingVisits: number;
  sessionId: string;
}

export interface APIError {
  error: string;
  code?: string;
  retryAfter?: number;
}
```

---

## 🧪 Tests

### Tests Unitaires (Component Testing)

```tsx
// frontend/components/letters/__tests__/AccessGate.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { AccessGate } from '../AccessGate';
import { apiClient } from '@/lib/api';

jest.mock('@/lib/api');

describe('AccessGate', () => {
  it('should show loading state initially', () => {
    render(
      <AccessGate>
        <div>Protected Content</div>
      </AccessGate>
    );
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('should show teaser when access denied', async () => {
    (apiClient.get as jest.Mock).mockResolvedValue({
      data: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
      },
    });

    render(
      <AccessGate>
        <div>Protected Content</div>
      </AccessGate>
    );

    await waitFor(() => {
      expect(screen.getByText(/Fonctionnalité Premium/i)).toBeInTheDocument();
      expect(screen.getByText(/1 \/ 3 visites/i)).toBeInTheDocument();
    });
  });

  it('should show children when access granted', async () => {
    (apiClient.get as jest.Mock).mockResolvedValue({
      data: {
        visitCount: 3,
        hasAccess: true,
        remainingVisits: 0,
      },
    });

    render(
      <AccessGate>
        <div>Protected Content</div>
      </AccessGate>
    );

    await waitFor(() => {
      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });
  });
});
```

```tsx
// frontend/components/letters/__tests__/LetterGenerator.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LetterGenerator } from '../LetterGenerator';
import { apiClient } from '@/lib/api';

jest.mock('@/lib/api');

describe('LetterGenerator', () => {
  it('should render form with company input', () => {
    render(<LetterGenerator />);
    expect(screen.getByLabelText(/Nom de l'entreprise/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Générer les lettres/i })).toBeInTheDocument();
  });

  it('should show validation error for empty input', async () => {
    render(<LetterGenerator />);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/au moins 2 caractères/i)).toBeInTheDocument();
    });
  });

  it('should call API and show preview on success', async () => {
    const mockResponse = {
      data: {
        id: '123',
        companyName: 'Google',
        motivationLetter: '# Motivation',
        antiMotivationLetter: '# Anti',
        createdAt: new Date().toISOString(),
      },
    };
    (apiClient.post as jest.Mock).mockResolvedValue(mockResponse);

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    fireEvent.change(input, { target: { value: 'Google' } });

    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/Lettres pour Google/i)).toBeInTheDocument();
    });
  });

  it('should show error message on 429 rate limit', async () => {
    (apiClient.post as jest.Mock).mockRejectedValue({
      response: {
        status: 429,
        headers: { 'retry-after': '120' },
        data: { error: 'Rate limit exceeded' },
      },
    });

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    fireEvent.change(input, { target: { value: 'Google' } });

    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/Limite atteinte/i)).toBeInTheDocument();
      expect(screen.getByText(/2 minutes/i)).toBeInTheDocument();
    });
  });
});
```

### Tests E2E (Playwright)

```typescript
// e2e/letters.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Letters Generator', () => {
  test('should show access gate on first visit', async ({ page, context }) => {
    // Clear cookies pour simuler 1ère visite
    await context.clearCookies();

    await page.goto('/letters');

    // Teaser doit être visible
    await expect(page.getByText('Fonctionnalité Premium')).toBeVisible();
    await expect(page.getByText(/0 \/ 3 visites/i)).toBeVisible();
  });

  test('should allow generation after 3 visits', async ({ page, context }) => {
    // Simuler 3 visites via cookies
    await context.addCookies([
      {
        name: 'visitor_session',
        value: 'test-session-id',
        domain: 'localhost',
        path: '/',
      },
    ]);

    // Mock Redis to return visitCount = 3
    // (nécessite mock au niveau backend ou fixture)

    await page.goto('/letters');

    // Form doit être visible
    await expect(page.getByLabel(/Nom de l'entreprise/i)).toBeVisible();

    // Remplir et soumettre
    await page.fill('input[name="companyName"]', 'Google');
    await page.click('button:has-text("Générer les lettres")');

    // Loading state
    await expect(page.getByText(/Génération en cours/i)).toBeVisible();

    // Preview après génération (timeout long pour IA)
    await expect(page.getByText(/Lettres pour Google/i)).toBeVisible({ timeout: 60000 });

    // Vérifier dual display
    await expect(page.getByText('Lettre de Motivation')).toBeVisible();
    await expect(page.getByText("Lettre d'Anti-Motivation")).toBeVisible();
  });

  test('should download PDF on button click', async ({ page, context }) => {
    // Setup: générer lettres d'abord
    await context.addCookies([
      { name: 'visitor_session', value: 'test-session-id', domain: 'localhost', path: '/' },
    ]);

    await page.goto('/letters');
    await page.fill('input[name="companyName"]', 'Microsoft');
    await page.click('button:has-text("Générer les lettres")');
    await page.waitForSelector('text=Lettres pour Microsoft');

    // Télécharger PDF dual
    const downloadPromise = page.waitForEvent('download');
    await page.click('button:has-text("PDF Dual")');
    const download = await downloadPromise;

    // Vérifier nom fichier
    expect(download.suggestedFilename()).toMatch(/lettre-Microsoft-both\.pdf/i);
  });

  test('should handle rate limit error gracefully', async ({ page, context }) => {
    // Mock API pour retourner 429
    await page.route('**/api/letters/generate', (route) => {
      route.fulfill({
        status: 429,
        headers: { 'retry-after': '120' },
        body: JSON.stringify({ error: 'Rate limit exceeded' }),
      });
    });

    await context.addCookies([
      { name: 'visitor_session', value: 'test-session-id', domain: 'localhost', path: '/' },
    ]);

    await page.goto('/letters');
    await page.fill('input[name="companyName"]', 'Amazon');
    await page.click('button:has-text("Générer les lettres")');

    // Message d'erreur
    await expect(page.getByText(/Limite atteinte/i)).toBeVisible();
    await expect(page.getByText(/2 minutes/i)).toBeVisible();
  });
});
```

### Commandes

```bash
# Tests unitaires
npm run test

# Tests E2E
npm run test:e2e

# Coverage
npm run test:coverage
```

---

## ⚠️ Points d'Attention

### 1. Security

- ⚠️ **Validation Input:** TOUJOURS valider côté serveur (ne jamais faire confiance au client)
- ⚠️ **Rate Limiting:** Double vérification (client + serveur) pour éviter abus
- ⚠️ **XSS Protection:** ReactMarkdown avec sanitization activée (remark-gfm safe)
- ⚠️ **CSRF:** Cookies SameSite=Lax minimum

### 2. Performance

- 💡 **Lazy Loading:** Charger LetterPreview uniquement si lettres générées
- 💡 **Debouncing:** Si auto-save dans localStorage, debouncer les writes
- 💡 **Memoization:** React.memo pour LetterPreview si re-renders fréquents
- ⚠️ **Memory Leaks:** Cleanup timers (progressInterval) dans useEffect return

### 3. UX

- ⚠️ **Loading States:** TOUJOURS afficher feedback pendant génération (30-60s)
- 💡 **Optimistic UI:** Afficher preview skeleton pendant chargement
- 💡 **Offline Handling:** Détecter offline et afficher message clair
- ⚠️ **Mobile UX:** Tester scroll dans lettres longues sur mobile

### 4. Edge Cases

- ⚠️ **Noms d'entreprise spéciaux:** Gérer accents, caractères spéciaux, espaces multiples
- ⚠️ **Lettres vides:** Que faire si IA retourne contenu vide ? (fallback message)
- ⚠️ **Session expirée:** Vérifier cookie session avant appel API (redirect login si nécessaire)
- ⚠️ **PDF trop longs:** Limiter taille lettres ou paginer PDF

### 5. Accessibility

- 💡 **Keyboard Navigation:** Tab entre champs, Enter pour submit
- 💡 **Screen Readers:** Labels ARIA pour loading states
- 💡 **Color Contrast:** Vérifier contraste vert/rouge pour daltoniens
- 💡 **Focus Management:** Focus input après reset

### 6. Cost Optimization (API IA)

- ⚠️ **Cache côté client:** Si même entreprise, proposer réutilisation
- ⚠️ **Avertissement coût:** Afficher "Génération restantes: X/5" pour sensibiliser
- 💡 **Throttle requests:** Minimum 2 minutes entre générations (cooldown)

---

## 📚 Ressources

### Documentation

- [React Hook Form](https://react-hook-form.com/) - Gestion formulaires
- [Zod](https://zod.dev/) - Validation schema
- [Framer Motion](https://www.framer.com/motion/) - Animations
- [react-markdown](https://github.com/remarkjs/react-markdown) - Rendu Markdown
- [Tailwind CSS](https://tailwindcss.com/) - Styling
- [Playwright](https://playwright.dev/) - Tests E2E

### Articles

- [Optimistic UI Patterns](https://www.smashingmagazine.com/2016/11/true-lies-of-optimistic-user-interfaces/)
- [Form Validation Best Practices](https://web.dev/sign-in-form-best-practices/)
- [Accessible Loading States](https://www.scottohara.me/blog/2018/01/18/loading-states.html)

### Outils

- [Lighthouse](https://developers.google.com/web/tools/lighthouse) - Audit performance
- [axe DevTools](https://www.deque.com/axe/devtools/) - Audit accessibilité
- [React DevTools](https://react.dev/learn/react-developer-tools) - Debugging

---

## ✅ Checklist de Complétion

### Code

- [ ] Page `/letters/page.tsx` créée avec metadata SEO
- [ ] Component `AccessGate` avec vérification API
- [ ] Component `LetterGenerator` avec form + validation Zod
- [ ] Component `LetterPreview` avec dual display responsive
- [ ] API client avec retry logic et error handling
- [ ] Types TypeScript pour toutes les interfaces
- [ ] Gestion états: loading, error, success

### Features

- [ ] Teaser si < 3 visites avec progression animée
- [ ] Form validation (client-side)
- [ ] Génération avec loading state + progress bar
- [ ] Affichage dual (2 colonnes) responsive
- [ ] Copy to clipboard (motivation + anti)
- [ ] Export PDF individuel (motivation, anti)
- [ ] Export PDF dual (les 2 lettres)
- [ ] Error handling (403, 429, 500)
- [ ] Historique dans localStorage
- [ ] Reset pour nouvelle génération

### Design

- [ ] Responsive (mobile, tablet, desktop)
- [ ] Dark mode support
- [ ] Animations Framer Motion (entrées, sorties)
- [ ] Distinction visuelle claire (vert vs rouge/orange)
- [ ] États hover/focus accessibles
- [ ] Loading skeletons

### Tests

- [ ] Tests unitaires AccessGate (3 cas)
- [ ] Tests unitaires LetterGenerator (4 cas)
- [ ] Tests unitaires LetterPreview (copy, download)
- [ ] Tests E2E flow complet (visite 1 → teaser)
- [ ] Tests E2E génération après 3 visites
- [ ] Tests E2E rate limit handling
- [ ] Tests E2E download PDF
- [ ] Coverage > 80%

### Documentation

- [ ] Comments dans code complexe
- [ ] JSDoc pour components publics
- [ ] README avec screenshots
- [ ] Storybook stories (optionnel)

### Performance

- [ ] Lazy loading components
- [ ] Image optimization (si screenshots)
- [ ] Bundle size < 50KB (analyze)
- [ ] Lighthouse score > 90

### Security

- [ ] Input sanitization (Zod regex)
- [ ] XSS protection (ReactMarkdown safe)
- [ ] Rate limiting UI feedback
- [ ] CORS headers vérifiés
- [ ] Cookies SameSite configurés

### Accessibility

- [ ] WCAG 2.1 AA compliance
- [ ] Keyboard navigation complète
- [ ] Screen reader testing (NVDA/JAWS)
- [ ] Color contrast ≥ 4.5:1
- [ ] Focus visible sur tous les interactifs

### Deployment

- [ ] Environment variables configurées (.env.example)
- [ ] Build réussi sans warnings
- [ ] Tests passent en CI
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
