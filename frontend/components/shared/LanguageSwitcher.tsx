'use client';

import { usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';

export function LanguageSwitcher() {
  const pathname = usePathname();

  // Extract current locale from pathname
  const currentLocale = pathname.startsWith('/en') ? 'en' : 'fr';
  const targetLocale = currentLocale === 'fr' ? 'en' : 'fr';

  const switchLocale = () => {
    // Remove current locale prefix from pathname
    const pathWithoutLocale = pathname.replace(/^\/(fr|en)/, '') || '/';
    // Build new URL with target locale
    const newPath = `/${targetLocale}${pathWithoutLocale}`;
    window.location.href = newPath;
  };

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={switchLocale}
      className="gap-2"
    >
      {targetLocale === 'en' ? (
        <>🇬🇧 EN</>
      ) : (
        <>🇫🇷 FR</>
      )}
    </Button>
  );
}
