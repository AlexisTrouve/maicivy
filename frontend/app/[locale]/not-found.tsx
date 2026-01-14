'use client';

import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';

export default function NotFound() {
  const t = useTranslations('errors');
  const tCommon = useTranslations('common');

  return (
    <div className="container flex min-h-screen flex-col items-center justify-center">
      <h1 className="text-6xl font-bold">404</h1>
      <p className="mt-4 text-xl text-muted-foreground">{t('notFound')}</p>
      <Button asChild className="mt-8">
        <Link href="/">{tCommon('backToHome')}</Link>
      </Button>
    </div>
  );
}
