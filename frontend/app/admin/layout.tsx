import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
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

// Polices — MÊME configuration que le site public (app/[locale]/layout.tsx).
//
// POURQUOI il faut les charger ICI aussi : ce fichier est une racine à part entière (il fournit son
// propre <html>/<body>, cf. commentaire ci-dessus), il n'hérite donc de rien du layout public.
//
// CE QUE ÇA CORRIGE — un défaut resté invisible jusqu'à ce qu'on regarde une capture : Tailwind pose
// `font-family: var(--font-inter), sans-serif`. Une var CSS NON DÉFINIE n'est pas ignorée au profit
// du terme suivant — elle rend la déclaration entière invalide, et la police retombe sur le défaut
// du navigateur. Tout le panneau admin était donc rendu en Times New Roman depuis toujours, sans que
// rien n'échoue nulle part : ni erreur, ni test rouge, juste un panneau qui avait l'air bon marché.
const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const poppins = Poppins({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-poppins',
  display: 'swap',
});

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  // Next 15 : cookies() est asynchrone, d'où le layout async.
  const cookieLocale = (await cookies()).get('NEXT_LOCALE')?.value;
  const locale = (locales as readonly string[]).includes(cookieLocale ?? '') ? cookieLocale! : defaultLocale;
  const messages = loadMessages(locale);

  return (
    <html lang={locale} suppressHydrationWarning>
      {/* admin-surface (globals.css) : fond profond + lueurs diluées. Remplace l'aplat bg-slate-950,
          qui donnait au panneau l'aspect d'un échafaudage plutôt que d'un produit. */}
      <body
        className={`${inter.variable} ${poppins.variable} admin-surface min-h-screen font-sans text-slate-100 antialiased`}
      >
        <NextIntlClientProvider locale={locale} messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
