'use client';

import { useState, useEffect } from 'react';
import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
// framer-motion — bibliothèque d'animation standard Next.js/React (pas 'motion/react')
import { motion } from 'framer-motion';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
// Code2 et Briefcase importés mais non utilisés ici — gardés pour extensions futures
import { FileText, Sparkles, BarChart3, Layers, ArrowRight, Code2, Briefcase } from 'lucide-react';

export default function HomePage() {
  const [mounted, setMounted] = useState(false);
  const t = useTranslations('home');

  useEffect(() => {
    setMounted(true);
  }, []);

  // Évite le flash de contenu non-stylé (FOUC) lié au theme dark/light
  if (!mounted) {
    return <div className="container py-12 md:py-24 animate-pulse"><div className="h-96"></div></div>;
  }

  return (
    <div className="container py-12 md:py-24">
      {/* ─── Hero section ─────────────────────────────────────────────── */}
      <div className="mx-auto max-w-4xl text-center">

        {/* Badge disponibilité freelance — point vert animé via animate-ping Tailwind */}
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="mb-6 inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary"
        >
          {/* Point vert animé — signale la disponibilité en temps réel */}
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-green-400" />
          </span>
          Disponible pour missions freelance
        </motion.div>

        {/* Identité — nom + rôle en uppercase discret au-dessus du titre principal */}
        <motion.p
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="mb-3 text-base font-medium text-muted-foreground tracking-widest uppercase"
        >
          Alexis Trouvé — Développeur Fullstack
        </motion.p>

        {/* Titre principal — gradient sur la deuxième moitié pour l'accroche visuelle.
            Split sur les mots : 2 premiers normaux, le reste en gradient bleu→cyan. */}
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="font-heading text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl"
        >
          {t('title').split(' ').slice(0, 2).join(' ')}{' '}
          <span className="bg-gradient-to-r from-primary via-blue-400 to-cyan-400 bg-clip-text text-transparent">
            {t('title').split(' ').slice(2).join(' ')}
          </span>
        </motion.h1>

        {/* Sous-titre — description concise de la valeur ajoutée */}
        <motion.p
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.35 }}
          className="mt-6 text-lg text-muted-foreground max-w-2xl mx-auto"
        >
          {t('subtitle')}
        </motion.p>

        {/* Stack technique — badges visuels avec backdrop-blur pour s'intégrer à l'aurora */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.45 }}
          className="mt-6 flex flex-wrap justify-center gap-2"
        >
          {['Go', 'Next.js', 'TypeScript', 'PostgreSQL', 'Redis', 'Docker'].map((tech) => (
            <span
              key={tech}
              className="inline-flex items-center rounded-md border border-border/60 bg-muted/40 px-2.5 py-1 text-xs font-medium text-muted-foreground backdrop-blur-sm"
            >
              {tech}
            </span>
          ))}
        </motion.div>

        {/* CTAs — bouton primaire avec shadow colorée, secondaire avec backdrop-blur */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.55 }}
          className="mt-10 flex flex-wrap justify-center gap-4"
        >
          <Button asChild size="lg" className="gap-2 shadow-lg shadow-primary/25">
            <Link href="/cv">
              <FileText className="h-4 w-4" />
              {t('cta.viewCV')}
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="gap-2 backdrop-blur-sm">
            <Link href="/letters">
              <Sparkles className="h-4 w-4" />
              {t('cta.generateLetter')}
            </Link>
          </Button>
        </motion.div>
      </div>

      {/* ─── Feature cards ────────────────────────────────────────────── */}
      {/* Délai 0.7s pour apparaître après le hero — donne un effet de cascade */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, delay: 0.7 }}
        className="mx-auto mt-24 grid max-w-5xl gap-6 md:grid-cols-2 lg:grid-cols-4"
      >
        {/* Card CV — backdrop-blur-sm s'intègre à l'aurora derrière */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <FileText className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.cv.title')}</CardTitle>
            <CardDescription>
              {t('features.cv.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/cv">
                {t('features.cv.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Lettres de motivation */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <Sparkles className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.letters.title')}</CardTitle>
            <CardDescription>
              {t('features.letters.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/letters">
                {t('features.letters.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Analytics */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <BarChart3 className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.analytics.title')}</CardTitle>
            <CardDescription>
              {t('features.analytics.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/analytics">
                {t('features.analytics.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Architecture — bordure en pointillés pour signaler que c'est différent/expérimental */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300 border-dashed border-primary/50">
          <CardHeader>
            <Layers className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.architecture.title')}</CardTitle>
            <CardDescription>
              {t('features.architecture.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/architecture">
                {t('features.architecture.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}
