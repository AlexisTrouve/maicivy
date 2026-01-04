'use client';

import { useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useTranslations('errors');

  useEffect(() => {
    // Log l'erreur (pourrait envoyer à un service de monitoring)
    console.error('Error boundary:', error);
  }, [error]);

  return (
    <div className="container flex min-h-screen items-center justify-center py-10">
      <Card className="max-w-md">
        <CardHeader>
          <CardTitle>{t('generic')}</CardTitle>
          <CardDescription>
            {t('tryAgain')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {process.env.NODE_ENV === 'development' && (
            <div className="rounded-md bg-muted p-4">
              <p className="text-sm font-mono text-muted-foreground">
                {error.message}
              </p>
            </div>
          )}
          <Button onClick={reset} className="w-full">
            {t('retry', { ns: 'common' })}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
