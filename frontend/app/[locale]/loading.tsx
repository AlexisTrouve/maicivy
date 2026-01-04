import { useTranslations } from 'next-intl';
import { LoadingSpinner } from '@/components/shared/LoadingSpinner';

export default function Loading() {
  const t = useTranslations('common');

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <LoadingSpinner size="lg" />
        <p className="mt-4 text-muted-foreground">{t('loading')}</p>
      </div>
    </div>
  );
}
