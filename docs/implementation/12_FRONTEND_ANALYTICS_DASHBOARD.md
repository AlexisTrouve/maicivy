# 12. FRONTEND_ANALYTICS_DASHBOARD

## 📋 Métadonnées

- **Phase:** 4
- **Priorité:** MOYENNE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 05. FRONTEND_FOUNDATION, 11. BACKEND_ANALYTICS
- **Temps estimé:** 4-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Créer un dashboard public d'analytics en temps réel permettant à tous les visiteurs de visualiser les statistiques d'utilisation du site : visiteurs actuels, thèmes CV les plus consultés, lettres générées, et interactions utilisateurs.

Ce dashboard démontre les compétences en visualisation de données, temps réel (WebSocket), et architecture moderne frontend.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────┐
│                  Page /analytics                         │
│                                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │  RealtimeVisitors (WebSocket)                     │  │
│  │  - Nombre visiteurs actuels                       │  │
│  │  - Auto-reconnect                                 │  │
│  └───────────────────────────────────────────────────┘  │
│                                                           │
│  ┌─────────────────────┐  ┌─────────────────────────┐  │
│  │  ThemeStats         │  │  LettersGenerated       │  │
│  │  - Bar chart        │  │  - Line chart           │  │
│  │  - Top 5 thèmes CV  │  │  - Évolution temporelle │  │
│  └─────────────────────┘  └─────────────────────────┘  │
│                                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Heatmap                                          │  │
│  │  - Carte cliquable interactions                   │  │
│  │  - Gradient de chaleur                            │  │
│  └───────────────────────────────────────────────────┘  │
│                                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Filters                                          │  │
│  │  - Date range picker                              │  │
│  │  - Groupement (jour/semaine/mois)                 │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘

Backend APIs:
├─ GET /api/analytics/realtime → WebSocket /ws/analytics
├─ GET /api/analytics/stats?period=day|week|month
├─ GET /api/analytics/themes
└─ GET /api/analytics/letters

Data Flow:
WebSocket → Real-time updates → RealtimeVisitors component
HTTP Polling (30s) → Stats agrégées → Charts components
```

### Design Decisions

**1. Visualisation Temps Réel**
- **WebSocket** pour visiteurs actuels (latence < 1s)
- **Polling HTTP 30s** pour stats agrégées (pas besoin temps réel strict)
- **Auto-reconnect** WebSocket si déconnexion

**2. Bibliothèque de Charts**
- **Chart.js** : Simple, performant, responsive
- Alternative D3.js : Plus flexible mais complexe (optionnel pour heatmap)
- **Recharts** : Alternative React-native si préféré

**3. Responsive Design**
- Grid adaptatif : 1 col (mobile), 2 cols (tablet), 3 cols (desktop)
- Charts redimensionnables automatiquement
- Touch-friendly pour mobile

**4. Animations**
- **Framer Motion** : Transitions entre filtres
- **CountUp** : Chiffres animés (effet compteur)
- Smooth transitions sur mise à jour données

---

## 📦 Dépendances

### Bibliothèques NPM

```bash
# Charts
npm install chart.js react-chartjs-2

# Alternative: Recharts
# npm install recharts

# WebSocket client
npm install socket.io-client

# Date picker
npm install react-day-picker date-fns

# CountUp animation
npm install react-countup

# Utilities
npm install clsx
```

### Types TypeScript

```bash
npm install --save-dev @types/chart.js
```

---

## 🔨 Implémentation

### Étape 1: Structure de Base - Page Analytics

**Description:** Créer la page principale `/analytics` avec layout responsive.

**Code: `frontend/app/analytics/page.tsx`**

```tsx
import { Metadata } from 'next';
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';
import ThemeStats from '@/components/analytics/ThemeStats';
import LettersGenerated from '@/components/analytics/LettersGenerated';
import Heatmap from '@/components/analytics/Heatmap';
import DateFilter from '@/components/analytics/DateFilter';
import { Suspense } from 'react';

export const metadata: Metadata = {
  title: 'Analytics - maicivy',
  description: 'Dashboard analytics en temps réel du CV interactif',
};

export default function AnalyticsPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Analytics Dashboard</h1>
        <p className="text-muted-foreground">
          Statistiques publiques en temps réel
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6">
        <DateFilter />
      </div>

      {/* Grid Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Realtime Visitors - Full width sur mobile, 1 col sur desktop */}
        <div className="lg:col-span-3">
          <Suspense fallback={<RealtimeVisitorsSkeleton />}>
            <RealtimeVisitors />
          </Suspense>
        </div>

        {/* Theme Stats - 2 cols sur desktop */}
        <div className="lg:col-span-2">
          <Suspense fallback={<StatsSkeleton />}>
            <ThemeStats />
          </Suspense>
        </div>

        {/* Letters Generated - 1 col sur desktop */}
        <div className="lg:col-span-1">
          <Suspense fallback={<StatsSkeleton />}>
            <LettersGenerated />
          </Suspense>
        </div>

        {/* Heatmap - Full width */}
        <div className="lg:col-span-3">
          <Suspense fallback={<HeatmapSkeleton />}>
            <Heatmap />
          </Suspense>
        </div>
      </div>
    </div>
  );
}

// Skeleton components pour loading states
function RealtimeVisitorsSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-24 bg-muted rounded" />
    </div>
  );
}

function StatsSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-64 bg-muted rounded" />
    </div>
  );
}

function HeatmapSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-96 bg-muted rounded" />
    </div>
  );
}
```

**Explications:**
- Layout responsive avec Grid Tailwind
- Suspense boundaries pour progressive loading
- Skeletons pendant chargement (UX améliorée)
- Metadata SEO pour page analytics

---

### Étape 2: RealtimeVisitors Component (WebSocket)

**Description:** Affiche le nombre de visiteurs actuels via WebSocket.

**Code: `frontend/components/analytics/RealtimeVisitors.tsx`**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import CountUp from 'react-countup';
import { Activity } from 'lucide-react';

interface RealtimeData {
  currentVisitors: number;
  timestamp: number;
}

export default function RealtimeVisitors() {
  const [visitors, setVisitors] = useState<number>(0);
  const [prevVisitors, setPrevVisitors] = useState<number>(0);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<Socket | null>(null);

  useEffect(() => {
    // WebSocket connection
    const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
    const newSocket = io(backendUrl, {
      path: '/ws/analytics',
      transports: ['websocket'],
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionAttempts: 5,
    });

    // Connection events
    newSocket.on('connect', () => {
      console.log('[WS] Connected to analytics');
      setIsConnected(true);
    });

    newSocket.on('disconnect', () => {
      console.log('[WS] Disconnected from analytics');
      setIsConnected(false);
    });

    // Realtime data updates
    newSocket.on('realtime', (data: RealtimeData) => {
      setPrevVisitors(visitors);
      setVisitors(data.currentVisitors);
    });

    // Error handling
    newSocket.on('error', (error) => {
      console.error('[WS] Error:', error);
    });

    setSocket(newSocket);

    // Cleanup on unmount
    return () => {
      newSocket.close();
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Activity className="h-5 w-5" />
          Visiteurs Actuels
        </h2>
        <div className="flex items-center gap-2">
          <div
            className={`h-2 w-2 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-red-500'
            } animate-pulse`}
          />
          <span className="text-xs text-muted-foreground">
            {isConnected ? 'En ligne' : 'Déconnecté'}
          </span>
        </div>
      </div>

      <div className="flex items-center justify-center py-8">
        <div className="text-center">
          <div className="text-6xl font-bold text-primary">
            <CountUp start={prevVisitors} end={visitors} duration={1} />
          </div>
          <p className="text-sm text-muted-foreground mt-2">
            {visitors === 0 || visitors === 1 ? 'personne' : 'personnes'} en ce moment
          </p>
        </div>
      </div>

      <div className="text-xs text-muted-foreground text-center">
        Mise à jour en temps réel via WebSocket
      </div>
    </div>
  );
}
```

**Explications:**
- **Socket.io** pour WebSocket client
- **Auto-reconnect** si déconnexion (jusqu'à 5 tentatives)
- **CountUp** pour animation du chiffre
- Indicateur de connexion (vert/rouge pulsé)
- Gestion des états : connected, disconnected, error

---

### Étape 3: ThemeStats Component (Bar Chart)

**Description:** Top 5 thèmes CV les plus consultés avec bar chart.

**Code: `frontend/components/analytics/ThemeStats.tsx`**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { Bar } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js';
import { BarChart3 } from 'lucide-react';

// Register Chart.js components
ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend);

interface ThemeStat {
  theme: string;
  count: number;
  percentage: number;
}

export default function ThemeStats() {
  const [stats, setStats] = useState<ThemeStat[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchThemeStats();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchThemeStats, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchThemeStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/analytics/themes`);

      if (!response.ok) throw new Error('Failed to fetch theme stats');

      const data = await response.json();
      setStats(data.themes || []);
    } catch (error) {
      console.error('Error fetching theme stats:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const chartData = {
    labels: stats.map((stat) => stat.theme),
    datasets: [
      {
        label: 'Vues',
        data: stats.map((stat) => stat.count),
        backgroundColor: 'rgba(59, 130, 246, 0.5)', // blue-500
        borderColor: 'rgba(59, 130, 246, 1)',
        borderWidth: 1,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      title: {
        display: false,
      },
      tooltip: {
        callbacks: {
          label: (context: any) => {
            const stat = stats[context.dataIndex];
            return `${context.parsed.y} vues (${stat.percentage.toFixed(1)}%)`;
          },
        },
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: {
          precision: 0,
        },
      },
    },
  };

  if (isLoading) {
    return <div className="rounded-lg border bg-card p-6 animate-pulse h-80" />;
  }

  return (
    <div className="rounded-lg border bg-card p-6">
      <h2 className="text-xl font-semibold flex items-center gap-2 mb-4">
        <BarChart3 className="h-5 w-5" />
        Top Thèmes CV
      </h2>

      <div className="h-64">
        <Bar data={chartData} options={chartOptions} />
      </div>

      <div className="mt-4 space-y-2">
        {stats.map((stat, index) => (
          <div key={stat.theme} className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-2">
              <span className="font-medium text-muted-foreground">#{index + 1}</span>
              <span className="font-medium">{stat.theme}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{stat.count} vues</span>
              <span className="text-xs bg-muted px-2 py-1 rounded">
                {stat.percentage.toFixed(1)}%
              </span>
            </div>
          </div>
        ))}
      </div>

      <div className="text-xs text-muted-foreground text-center mt-4">
        Mise à jour toutes les 30 secondes
      </div>
    </div>
  );
}
```

**Explications:**
- **Chart.js** avec React wrapper `react-chartjs-2`
- **Auto-refresh 30s** (polling HTTP)
- Bar chart responsive avec tooltip custom
- Liste détaillée sous le graphique (ranking)
- Pourcentages affichés

---

### Étape 4: LettersGenerated Component (Line Chart)

**Description:** Évolution du nombre de lettres générées dans le temps.

**Code: `frontend/components/analytics/LettersGenerated.tsx`**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { Line } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { FileText } from 'lucide-react';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

interface LettersStat {
  date: string;
  count: number;
}

type Period = 'day' | 'week' | 'month';

export default function LettersGenerated() {
  const [stats, setStats] = useState<LettersStat[]>([]);
  const [period, setPeriod] = useState<Period>('week');
  const [totalLetters, setTotalLetters] = useState(0);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchLettersStats();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchLettersStats, 30000);
    return () => clearInterval(interval);
  }, [period]);

  const fetchLettersStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
      const response = await fetch(
        `${backendUrl}/api/analytics/letters?period=${period}`
      );

      if (!response.ok) throw new Error('Failed to fetch letters stats');

      const data = await response.json();
      setStats(data.history || []);
      setTotalLetters(data.total || 0);
    } catch (error) {
      console.error('Error fetching letters stats:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const chartData = {
    labels: stats.map((stat) => {
      const date = new Date(stat.date);
      return date.toLocaleDateString('fr-FR', {
        day: '2-digit',
        month: 'short',
      });
    }),
    datasets: [
      {
        label: 'Lettres générées',
        data: stats.map((stat) => stat.count),
        borderColor: 'rgba(34, 197, 94, 1)', // green-500
        backgroundColor: 'rgba(34, 197, 94, 0.1)',
        fill: true,
        tension: 0.3,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: {
          precision: 0,
        },
      },
    },
  };

  if (isLoading) {
    return <div className="rounded-lg border bg-card p-6 animate-pulse h-80" />;
  }

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <FileText className="h-5 w-5" />
          Lettres IA
        </h2>

        {/* Period selector */}
        <div className="flex gap-1 bg-muted p-1 rounded-md">
          {(['day', 'week', 'month'] as Period[]).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-3 py-1 text-xs rounded transition-colors ${
                period === p
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-background'
              }`}
            >
              {p === 'day' ? 'Jour' : p === 'week' ? 'Semaine' : 'Mois'}
            </button>
          ))}
        </div>
      </div>

      {/* Total counter */}
      <div className="mb-4 text-center">
        <div className="text-3xl font-bold text-primary">{totalLetters}</div>
        <div className="text-xs text-muted-foreground">lettres générées au total</div>
      </div>

      {/* Line chart */}
      <div className="h-48">
        <Line data={chartData} options={chartOptions} />
      </div>

      <div className="text-xs text-muted-foreground text-center mt-4">
        Évolution sur {period === 'day' ? '24h' : period === 'week' ? '7 jours' : '30 jours'}
      </div>
    </div>
  );
}
```

**Explications:**
- Line chart avec **area fill** (gradient sous la courbe)
- **Sélecteur de période** (jour/semaine/mois)
- **Total counter** affiché en haut
- Smooth curve (tension: 0.3)
- Auto-refresh 30s

---

### Étape 5: Heatmap Component

**Description:** Carte de chaleur des interactions utilisateurs.

**Code: `frontend/components/analytics/Heatmap.tsx`**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { Flame } from 'lucide-react';

interface HeatmapPoint {
  x: number;
  y: number;
  intensity: number;
}

interface HeatmapData {
  points: HeatmapPoint[];
  maxIntensity: number;
}

export default function Heatmap() {
  const [heatmapData, setHeatmapData] = useState<HeatmapData>({
    points: [],
    maxIntensity: 0,
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchHeatmapData();

    // Auto-refresh every 60 seconds
    const interval = setInterval(fetchHeatmapData, 60000);
    return () => clearInterval(interval);
  }, []);

  const fetchHeatmapData = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/analytics/heatmap`);

      if (!response.ok) throw new Error('Failed to fetch heatmap data');

      const data = await response.json();
      setHeatmapData(data);
    } catch (error) {
      console.error('Error fetching heatmap data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const getHeatColor = (intensity: number, max: number): string => {
    if (max === 0) return 'rgba(59, 130, 246, 0.1)'; // blue-500 very light

    const ratio = intensity / max;

    // Gradient bleu → vert → jaune → rouge
    if (ratio < 0.25) {
      return `rgba(59, 130, 246, ${0.2 + ratio * 2})`; // blue
    } else if (ratio < 0.5) {
      return `rgba(34, 197, 94, ${0.3 + ratio * 2})`; // green
    } else if (ratio < 0.75) {
      return `rgba(234, 179, 8, ${0.4 + ratio * 2})`; // yellow
    } else {
      return `rgba(239, 68, 68, ${0.5 + ratio * 2})`; // red
    }
  };

  if (isLoading) {
    return <div className="rounded-lg border bg-card p-6 animate-pulse h-96" />;
  }

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Flame className="h-5 w-5" />
          Heatmap des Interactions
        </h2>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Faible</span>
          <div className="flex gap-1">
            <div className="w-6 h-4 rounded" style={{ backgroundColor: 'rgba(59, 130, 246, 0.5)' }} />
            <div className="w-6 h-4 rounded" style={{ backgroundColor: 'rgba(34, 197, 94, 0.6)' }} />
            <div className="w-6 h-4 rounded" style={{ backgroundColor: 'rgba(234, 179, 8, 0.7)' }} />
            <div className="w-6 h-4 rounded" style={{ backgroundColor: 'rgba(239, 68, 68, 0.8)' }} />
          </div>
          <span>Fort</span>
        </div>
      </div>

      {/* Heatmap Grid */}
      <div className="relative w-full h-80 bg-muted/20 rounded-lg overflow-hidden">
        {heatmapData.points.map((point, index) => (
          <div
            key={index}
            className="absolute rounded-full blur-xl transition-all duration-500"
            style={{
              left: `${point.x}%`,
              top: `${point.y}%`,
              width: '100px',
              height: '100px',
              transform: 'translate(-50%, -50%)',
              backgroundColor: getHeatColor(point.intensity, heatmapData.maxIntensity),
              pointerEvents: 'none',
            }}
          />
        ))}

        {/* Overlay grid for reference */}
        <div className="absolute inset-0 grid grid-cols-10 grid-rows-10 pointer-events-none opacity-10">
          {Array.from({ length: 100 }).map((_, i) => (
            <div key={i} className="border border-foreground" />
          ))}
        </div>

        {heatmapData.points.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
            Aucune donnée d'interaction disponible
          </div>
        )}
      </div>

      <div className="text-xs text-muted-foreground text-center mt-4">
        Basé sur {heatmapData.points.length} points d'interaction
      </div>
    </div>
  );
}
```

**Explications:**
- **Heatmap basique** avec divs positionnées (blur pour effet de chaleur)
- **Gradient de couleur** : bleu → vert → jaune → rouge selon intensité
- Alternative avancée : D3.js heatmap (plus complexe)
- Grille de référence 10x10 en overlay
- Auto-refresh 60s (moins fréquent car moins critique)

---

### Étape 6: DateFilter Component

**Description:** Filtre de plage de dates pour toutes les statistiques.

**Code: `frontend/components/analytics/DateFilter.tsx`**

```tsx
'use client';

import { useState } from 'react';
import { Calendar } from 'lucide-react';
import { DayPicker } from 'react-day-picker';
import 'react-day-picker/dist/style.css';
import { format } from 'date-fns';
import { fr } from 'date-fns/locale';

interface DateRange {
  from: Date | undefined;
  to: Date | undefined;
}

export default function DateFilter() {
  const [dateRange, setDateRange] = useState<DateRange>({
    from: undefined,
    to: undefined,
  });
  const [isOpen, setIsOpen] = useState(false);

  const handleSelect = (range: DateRange | undefined) => {
    setDateRange(range || { from: undefined, to: undefined });
  };

  const formatDateRange = (): string => {
    if (!dateRange.from) return 'Sélectionner une période';

    if (!dateRange.to) {
      return format(dateRange.from, 'dd MMM yyyy', { locale: fr });
    }

    return `${format(dateRange.from, 'dd MMM', { locale: fr })} - ${format(
      dateRange.to,
      'dd MMM yyyy',
      { locale: fr }
    )}`;
  };

  return (
    <div className="relative inline-block">
      {/* Trigger button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-4 py-2 border rounded-lg hover:bg-accent transition-colors"
      >
        <Calendar className="h-4 w-4" />
        <span className="text-sm">{formatDateRange()}</span>
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute top-full mt-2 z-50 bg-card border rounded-lg shadow-lg p-4">
          <DayPicker
            mode="range"
            selected={dateRange}
            onSelect={handleSelect}
            locale={fr}
            numberOfMonths={2}
          />

          <div className="flex gap-2 mt-4 pt-4 border-t">
            <button
              onClick={() => {
                setDateRange({ from: undefined, to: undefined });
                setIsOpen(false);
              }}
              className="flex-1 px-3 py-2 text-sm border rounded hover:bg-accent"
            >
              Réinitialiser
            </button>
            <button
              onClick={() => setIsOpen(false)}
              className="flex-1 px-3 py-2 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90"
            >
              Appliquer
            </button>
          </div>
        </div>
      )}

      {/* Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}
    </div>
  );
}
```

**Explications:**
- **react-day-picker** pour sélecteur de dates
- Mode **range** (plage de dates)
- Locale française (date-fns/locale)
- Dropdown avec backdrop (fermeture au clic extérieur)
- Boutons réinitialiser/appliquer

---

### Étape 7: API Client Utility

**Description:** Wrapper API pour simplifier les appels analytics.

**Code: `frontend/lib/analytics-api.ts`**

```typescript
const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';

export interface AnalyticsStats {
  period: 'day' | 'week' | 'month';
  totalVisitors: number;
  totalPageViews: number;
  avgSessionDuration: number;
}

export interface ThemeStat {
  theme: string;
  count: number;
  percentage: number;
}

export interface LettersStat {
  date: string;
  count: number;
}

export interface LettersResponse {
  total: number;
  history: LettersStat[];
}

export interface HeatmapPoint {
  x: number;
  y: number;
  intensity: number;
}

export interface HeatmapData {
  points: HeatmapPoint[];
  maxIntensity: number;
}

// Fetch general stats
export async function fetchAnalyticsStats(
  period: 'day' | 'week' | 'month' = 'week'
): Promise<AnalyticsStats> {
  const response = await fetch(`${BACKEND_URL}/api/analytics/stats?period=${period}`);

  if (!response.ok) {
    throw new Error(`Failed to fetch analytics stats: ${response.statusText}`);
  }

  return response.json();
}

// Fetch theme stats
export async function fetchThemeStats(): Promise<{ themes: ThemeStat[] }> {
  const response = await fetch(`${BACKEND_URL}/api/analytics/themes`);

  if (!response.ok) {
    throw new Error(`Failed to fetch theme stats: ${response.statusText}`);
  }

  return response.json();
}

// Fetch letters stats
export async function fetchLettersStats(
  period: 'day' | 'week' | 'month' = 'week'
): Promise<LettersResponse> {
  const response = await fetch(`${BACKEND_URL}/api/analytics/letters?period=${period}`);

  if (!response.ok) {
    throw new Error(`Failed to fetch letters stats: ${response.statusText}`);
  }

  return response.json();
}

// Fetch heatmap data
export async function fetchHeatmapData(): Promise<HeatmapData> {
  const response = await fetch(`${BACKEND_URL}/api/analytics/heatmap`);

  if (!response.ok) {
    throw new Error(`Failed to fetch heatmap data: ${response.statusText}`);
  }

  return response.json();
}
```

**Explications:**
- Types TypeScript pour toutes les réponses API
- Functions réutilisables pour chaque endpoint
- Error handling centralisé
- Configuration BACKEND_URL via env var

---

## 🧪 Tests

### Tests Unitaires (Jest + React Testing Library)

**Code: `frontend/components/analytics/__tests__/RealtimeVisitors.test.tsx`**

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import RealtimeVisitors from '../RealtimeVisitors';
import { io } from 'socket.io-client';

// Mock socket.io-client
jest.mock('socket.io-client');

const mockSocket = {
  on: jest.fn(),
  close: jest.fn(),
};

(io as jest.Mock).mockReturnValue(mockSocket);

describe('RealtimeVisitors', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render with initial state', () => {
    render(<RealtimeVisitors />);

    expect(screen.getByText('Visiteurs Actuels')).toBeInTheDocument();
    expect(screen.getByText('Déconnecté')).toBeInTheDocument();
  });

  it('should connect to WebSocket on mount', () => {
    render(<RealtimeVisitors />);

    expect(io).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        path: '/ws/analytics',
        transports: ['websocket'],
      })
    );
  });

  it('should update visitors count on realtime event', async () => {
    render(<RealtimeVisitors />);

    // Simulate WebSocket event
    const onRealtimeCallback = mockSocket.on.mock.calls.find(
      (call) => call[0] === 'realtime'
    )?.[1];

    onRealtimeCallback?.({ currentVisitors: 42, timestamp: Date.now() });

    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument();
    });
  });

  it('should show connected status when connected', () => {
    render(<RealtimeVisitors />);

    const onConnectCallback = mockSocket.on.mock.calls.find(
      (call) => call[0] === 'connect'
    )?.[1];

    onConnectCallback?.();

    expect(screen.getByText('En ligne')).toBeInTheDocument();
  });

  it('should cleanup on unmount', () => {
    const { unmount } = render(<RealtimeVisitors />);

    unmount();

    expect(mockSocket.close).toHaveBeenCalled();
  });
});
```

### Tests E2E (Playwright)

**Code: `frontend/e2e/analytics.spec.ts`**

```typescript
import { test, expect } from '@playwright/test';

test.describe('Analytics Dashboard', () => {
  test('should load analytics page', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('h1')).toContainText('Analytics Dashboard');
  });

  test('should display realtime visitors', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('text=Visiteurs Actuels')).toBeVisible();
    await expect(page.locator('text=En ligne').or(page.locator('text=Déconnecté'))).toBeVisible();
  });

  test('should display theme stats chart', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('text=Top Thèmes CV')).toBeVisible();

    // Wait for chart to load
    await page.waitForSelector('canvas', { timeout: 5000 });
  });

  test('should allow period selection for letters stats', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('text=Lettres IA')).toBeVisible();

    // Click on "Mois" button
    await page.click('button:has-text("Mois")');

    // Verify button is active
    await expect(page.locator('button:has-text("Mois")')).toHaveClass(/bg-primary/);
  });

  test('should display heatmap', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('text=Heatmap des Interactions')).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/analytics');

    // Components should stack vertically on mobile
    await expect(page.locator('text=Analytics Dashboard')).toBeVisible();
    await expect(page.locator('text=Visiteurs Actuels')).toBeVisible();
  });
});
```

### Commandes

```bash
# Run unit tests
npm run test

# Run E2E tests
npm run test:e2e

# Run tests in watch mode
npm run test:watch

# Coverage report
npm run test:coverage
```

---

## ⚠️ Points d'Attention

### 1. WebSocket Reconnection

- **Problème:** Déconnexions fréquentes si serveur redémarre
- **Solution:** Implémentation auto-reconnect avec backoff exponentiel
- **Code:** `reconnectionDelay: 1000, reconnectionAttempts: 5`

### 2. Performance Charts

- **Problème:** Charts lourds avec beaucoup de données
- **Solution:** Limiter nombre de points affichés (max 30 jours)
- **Optimisation:** `maintainAspectRatio: false` pour meilleure performance

### 3. Memory Leaks

- **Problème:** Intervals et WebSocket non nettoyés
- **Solution:** Toujours cleanup dans `useEffect` return
- **Pattern:**
```tsx
useEffect(() => {
  const interval = setInterval(...);
  return () => clearInterval(interval);
}, []);
```

### 4. CORS WebSocket

- **Problème:** WebSocket bloqué par CORS en production
- **Solution:** Backend doit autoriser origin frontend
- **Backend config (Go):**
```go
router.Use(cors.New(cors.Config{
  AllowOrigins:     []string{"https://maicivy.com"},
  AllowMethods:     []string{"GET", "POST"},
  AllowWebSockets:  true,
}))
```

### 5. Date Timezone

- **Problème:** Dates inconsistentes selon timezone client
- **Solution:** Toujours utiliser UTC côté backend, convertir côté frontend
- **Code:**
```tsx
const date = new Date(stat.date);
date.toLocaleDateString('fr-FR', { timeZone: 'Europe/Paris' });
```

### 6. Chart.js Tree Shaking

- **Problème:** Bundle size élevé si tous les components Chart.js importés
- **Solution:** Importer UNIQUEMENT les components nécessaires
- **Bon pattern:**
```tsx
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  // N'importer que ce qui est utilisé
} from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, BarElement);
```

### 7. Heatmap Performance

- **Problème:** Trop de points ralentit le rendu
- **Solution:** Limiter à 100 points max, agréger si nécessaire
- **Backend:** Grouper points proches pour réduire volume de données

### 8. Loading States

- **Problème:** Flash de contenu vide avant chargement
- **Solution:** Skeletons systematiques avec Suspense
- **Pattern:** Toujours `<Suspense fallback={<Skeleton />}>`

---

## 📚 Ressources

### Documentation Officielle

- [Chart.js Documentation](https://www.chartjs.org/docs/latest/)
- [React Chart.js 2](https://react-chartjs-2.js.org/)
- [Socket.io Client](https://socket.io/docs/v4/client-api/)
- [React Day Picker](https://react-day-picker.js.org/)
- [Framer Motion](https://www.framer.com/motion/)

### Tutoriels

- [Building Real-time Dashboards with WebSocket](https://www.builder.io/blog/realtime-dashboard)
- [Chart.js Best Practices](https://www.chartjs.org/docs/latest/general/performance.html)
- [Next.js App Router Data Fetching](https://nextjs.org/docs/app/building-your-application/data-fetching)

### Alternatives

**Charts Libraries:**
- **Recharts** : Alternative React-native à Chart.js
- **Victory** : Composants modulaires, bonne accessibilité
- **D3.js** : Maximum flexibilité, courbe d'apprentissage élevée

**Heatmap Libraries:**
- **h337.js (heatmap.js)** : Library dédiée heatmap
- **D3-contour** : Contour plots avec D3
- **Plotly.js** : Heatmap 2D interactive

---

## ✅ Checklist de Complétion

### Code

- [ ] Page `/analytics` créée avec layout responsive
- [ ] `RealtimeVisitors` component avec WebSocket
- [ ] `ThemeStats` component avec Bar Chart
- [ ] `LettersGenerated` component avec Line Chart
- [ ] `Heatmap` component implémenté
- [ ] `DateFilter` component fonctionnel
- [ ] API client utilities (`analytics-api.ts`)
- [ ] Gestion des loading states (Suspense, skeletons)
- [ ] Gestion des error states

### Tests

- [ ] Tests unitaires `RealtimeVisitors`
- [ ] Tests unitaires `ThemeStats`
- [ ] Tests unitaires `LettersGenerated`
- [ ] Tests unitaires `Heatmap`
- [ ] Tests E2E (Playwright) : navigation analytics
- [ ] Tests E2E : vérification charts
- [ ] Tests E2E : responsive mobile
- [ ] Coverage > 70%

### UX

- [ ] Animations smooth (Framer Motion)
- [ ] CountUp effect sur chiffres
- [ ] Auto-refresh configuré (30s/60s)
- [ ] WebSocket auto-reconnect
- [ ] Responsive mobile/tablet/desktop
- [ ] Dark mode supporté
- [ ] Loading skeletons élégants
- [ ] Messages d'erreur clairs

### Performance

- [ ] Tree shaking Chart.js (imports sélectifs)
- [ ] Lazy loading components (dynamic imports)
- [ ] Memoization si nécessaire (React.memo)
- [ ] Cleanup intervals/WebSocket
- [ ] Limitation nombre de points charts
- [ ] Bundle size < 300KB pour page analytics

### Documentation

- [ ] Commentaires code (JSDoc)
- [ ] README component analytics
- [ ] Storybook stories (optionnel)
- [ ] Types TypeScript complets

### Integration

- [ ] Connection backend analytics API
- [ ] Variables d'environnement configurées
- [ ] CORS configuré pour WebSocket
- [ ] Health check WebSocket
- [ ] Tests integration backend ↔ frontend

### Sécurité

- [ ] Validation données reçues (types guards)
- [ ] Sanitization données affichées
- [ ] Rate limiting côté client (éviter spam)
- [ ] Protection XSS (React escape automatique)

### Déploiement

- [ ] Build production sans erreurs
- [ ] Lighthouse score > 90
- [ ] Vérification responsive (BrowserStack)
- [ ] Tests cross-browser (Chrome, Firefox, Safari)
- [ ] Meta tags SEO

### Qualité

- [ ] ESLint sans warnings
- [ ] Prettier formatting
- [ ] TypeScript strict mode
- [ ] Review sécurité
- [ ] Review performance
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
