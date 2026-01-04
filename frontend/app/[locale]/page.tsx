'use client';

import { useState, useEffect } from 'react';
import { Link } from '@/i18n/navigation';
import NextLink from 'next/link';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileText, Sparkles, BarChart3, Layers, Box } from 'lucide-react';

export default function HomePage() {
  const [mounted, setMounted] = useState(false);
  const t = useTranslations('home');

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return <div className="container py-12 md:py-24 animate-pulse"><div className="h-96"></div></div>;
  }

  return (
    <div className="container py-12 md:py-24">
      <div className="mx-auto max-w-4xl text-center">
        <h1 className="font-heading text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl">
          {t('title')}
        </h1>
        <p className="mt-6 text-lg text-muted-foreground">
          {t('subtitle')}
        </p>

        <div className="mt-10 flex flex-wrap justify-center gap-4">
          <Button asChild size="lg">
            <Link href="/cv">{t('cta.viewCV')}</Link>
          </Button>
          <Button asChild size="lg" variant="outline">
            <Link href="/letters">{t('cta.generateLetter')}</Link>
          </Button>
        </div>

        <div className="mt-6">
          <Button asChild variant="ghost" className="gap-2">
            <NextLink href="/3d-demo">
              <Box className="h-4 w-4" />
              Portfolio 3D
            </NextLink>
          </Button>
        </div>
      </div>

      <div className="mx-auto mt-24 grid max-w-5xl gap-8 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader>
            <FileText className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.cv.title')}</CardTitle>
            <CardDescription>
              {t('features.cv.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/cv">{t('features.cv.action')}</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <Sparkles className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.letters.title')}</CardTitle>
            <CardDescription>
              {t('features.letters.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/letters">{t('features.letters.action')}</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <BarChart3 className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.analytics.title')}</CardTitle>
            <CardDescription>
              {t('features.analytics.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/analytics">{t('features.analytics.action')}</Link>
            </Button>
          </CardContent>
        </Card>

        <Card className="border-dashed border-primary/50">
          <CardHeader>
            <Layers className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.architecture.title')}</CardTitle>
            <CardDescription>
              {t('features.architecture.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/architecture">{t('features.architecture.action')}</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
