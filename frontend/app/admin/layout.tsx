import type { Metadata } from 'next';
import { cookies } from 'next/headers';
import { NextIntlClientProvider } from 'next-intl';
import { loadMessages } from '@/i18n/messages';
import { locales, defaultLocale } from '@/i18n/config';
import '../globals.css';

// Panneau owner — hors du site i18n public (route top-level, fournit son propre <html>/<body>
// puisque app/layout.tsx est un pass-through). JAMAIS indexé : noindex/nofollow + robots.txt Disallow.
//
// i18n : l'admin n'a pas de préfixe de locale dans l'URL → on lit la langue dans le cookie NEXT_LOCALE
// (le même que le site public + le sélecteur de l'admin) et on fournit un NextIntlClientProvider à
// tous les composants client du panneau. router.refresh() après changement de cookie re-rend ce layout.
export const metadata: Metadata = {
  title: 'Admin · maicivy',
  robots: { index: false, follow: false },
};

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const cookieLocale = cookies().get('NEXT_LOCALE')?.value;
  const locale = (locales as readonly string[]).includes(cookieLocale ?? '') ? cookieLocale! : defaultLocale;
  const messages = loadMessages(locale);

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className="min-h-screen bg-slate-950 text-slate-100 antialiased">
        <NextIntlClientProvider locale={locale} messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
