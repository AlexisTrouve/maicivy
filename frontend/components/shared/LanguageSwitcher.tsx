'use client';

import { useLocale } from 'next-intl';
import { useRouter, usePathname } from '@/i18n/navigation';
import { Button } from '@/components/ui/button';

export function LanguageSwitcher() {
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  const switchLocale = (newLocale: 'fr' | 'en') => {
    router.replace(pathname, { locale: newLocale });
  };

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => switchLocale(locale === 'fr' ? 'en' : 'fr')}
      className="gap-2"
    >
      {locale === 'fr' ? (
        <>🇬🇧 EN</>
      ) : (
        <>🇫🇷 FR</>
      )}
    </Button>
  );
}
