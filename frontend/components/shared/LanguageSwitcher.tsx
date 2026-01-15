'use client';

import { useLocale } from 'next-intl';
import { usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';

export function LanguageSwitcher() {
  const locale = useLocale();
  const pathname = usePathname();

  const switchLocale = (newLocale: 'fr' | 'en') => {
    // Remove current locale prefix from pathname
    const pathWithoutLocale = pathname.replace(/^\/(fr|en)/, '') || '/';
    // Build new URL with new locale
    const newPath = `/${newLocale}${pathWithoutLocale}`;
    window.location.href = newPath;
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
