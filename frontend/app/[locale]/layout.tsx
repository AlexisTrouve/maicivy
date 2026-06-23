import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import { NextIntlClientProvider, hasLocale } from 'next-intl';
import { getMessages } from 'next-intl/server';
import { notFound } from 'next/navigation';
import Script from 'next/script';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { VisitorHeartbeatProvider } from '@/components/providers/VisitorHeartbeatProvider';
import { ThemeProvider } from '@/components/providers/ThemeProvider';
import { BackgroundProvider } from '@/components/background/BackgroundProvider';
import { BackgroundHost } from '@/components/background/BackgroundHost';
import ClickTracker from '@/components/analytics/ClickTracker';
import { locales } from '@/i18n/config';
import { loadMessages } from '@/i18n/messages';
import '../globals.css';

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

// Generate dynamic metadata based on locale
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }> | { locale: string };
}): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale;
  // QUOI : rejette toute locale hors liste blanche (`locales`) → vrai 404.
  // POURQUOI : un scanner tape /aws-credentials.json → Next matche [locale]="aws-credentials.json"
  //   (segment unique avec point, qui bypasse le middleware i18n via le matcher `.*\..*`). Sans
  //   cette garde, le layout rendait la homepage en HTTP 200 (fallback silencieux sur 'fr' dans
  //   i18n/request.ts) → faux signal de succès pour le scanner + 0 incrément du score sus (qui ne
  //   bump que sur 4xx). notFound() ici coupe l'exécution AVANT loadMessages ci-dessous → une locale
  //   hors liste blanche ne charge jamais de messages (et ne rend pas un soft-200 trompeur).
  // COMMENT : hasLocale (next-intl) teste l'appartenance à `locales` ; sinon notFound() lève
  //   NEXT_NOT_FOUND → rendu de app/not-found.tsx (racine, sans i18n) avec un statut 404 réel.
  if (!hasLocale(locales, locale)) {
    notFound();
  }
  const messages = loadMessages(locale);

  const baseUrl = process.env.NEXT_PUBLIC_BASE_URL || 'https://maicivy.etheryale.com';

  return {
    title: {
      default: messages.metadata.title,
      template: '%s | maicivy',
    },
    description: messages.metadata.description,
    keywords: ['CV', 'portfolio', 'AI', 'developer', 'full-stack', 'IA', 'développeur'],
    authors: [{ name: 'Alexis Trouvé' }],
    creator: 'Alexis Trouvé',
    metadataBase: new URL(baseUrl),
    openGraph: {
      type: 'website',
      locale: locale === 'fr' ? 'fr_FR' : 'en_US',
      url: baseUrl,
      siteName: 'maicivy',
      title: messages.metadata.title,
      description: messages.metadata.description,
      images: [
        {
          // Route OG dynamique Next.js — génère l'image 1200×630 via @vercel/og (ImageResponse)
          url: `${baseUrl}/api/og?locale=${locale}`,
          width: 1200,
          height: 630,
          alt: messages.metadata.title,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: messages.metadata.title,
      description: messages.metadata.description,
      // Route OG dynamique — même URL que openGraph.images
      images: [`${baseUrl}/api/og?locale=${locale}`],
      creator: '@AlexisTrouve',
    },
    robots: {
      index: true,
      follow: true,
    },
  };
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }> | { locale: string };
}) {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale;
  // Voir generateMetadata pour le QUOI/POURQUOI/COMMENT. Le layout est le point DRY qui enveloppe
  // toutes les pages sous [locale] → valider ici protège chaque route. notFound() AVANT getMessages
  // (sinon on chargerait des messages pour une locale qui n'existe pas).
  if (!hasLocale(locales, locale)) {
    notFound();
  }
  const messages = await getMessages({ locale });

  // data-URI SVG noise inline — évite un réseau request supplémentaire
  const noiseSvg = "data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E";

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className={`${inter.variable} ${poppins.variable} font-sans antialiased bg-background text-foreground`}>
        <Script id="theme-script" strategy="beforeInteractive">
          {`
            (function() {
              try {
                const theme = localStorage.getItem('theme');
                const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                const initialTheme = theme || (prefersDark ? 'dark' : 'light');
                if (initialTheme === 'dark') {
                  document.documentElement.classList.add('dark');
                } else {
                  document.documentElement.classList.remove('dark');
                }
              } catch (e) {}
            })();
          `}
        </Script>
        <NextIntlClientProvider messages={messages}>
          <ThemeProvider>
            <VisitorHeartbeatProvider showActiveVisitors={false}>
              {/* BackgroundProvider — état de sélection du fond animé, partagé entre la coquille
                  (BackgroundHost ci-dessous) et le sélecteur (BackgroundSwitcher, dans le Header). */}
              <BackgroundProvider>
              <div className="relative flex min-h-screen flex-col bg-background overflow-hidden">
                {/* Aurora background — blobs lumineux animés positionnés en fixed z-0.
                    Visible sur toutes les pages, derrière tout le contenu.
                    Les opacités dark: sont plus hautes pour compenser le fond très sombre. */}
                <div aria-hidden="true" className="pointer-events-none fixed inset-0 z-0 overflow-hidden">
                  {/* Blob 1 — bleu primaire, haut gauche — animation 18s */}
                  <div
                    className="absolute -top-[20%] -left-[10%] w-[60vw] h-[60vw] max-w-[700px] max-h-[700px] rounded-full opacity-[0.15] dark:opacity-[0.18] blur-[100px]"
                    style={{
                      background: 'radial-gradient(circle, hsl(217.2 91.2% 59.8%) 0%, transparent 70%)',
                      animation: 'aurora-1 18s ease-in-out infinite',
                    }}
                  />
                  {/* Blob 2 — violet/indigo, bas droite — animation 22s (désynchronisé du blob 1) */}
                  <div
                    className="absolute -bottom-[20%] -right-[10%] w-[55vw] h-[55vw] max-w-[650px] max-h-[650px] rounded-full opacity-[0.12] dark:opacity-[0.15] blur-[120px]"
                    style={{
                      background: 'radial-gradient(circle, hsl(262 83% 58%) 0%, transparent 70%)',
                      animation: 'aurora-2 22s ease-in-out infinite',
                    }}
                  />
                  {/* Blob 3 — cyan/teal, centre droite — animation 26s (désynchronisé) */}
                  <div
                    className="absolute top-[30%] -right-[5%] w-[40vw] h-[40vw] max-w-[500px] max-h-[500px] rounded-full opacity-[0.08] dark:opacity-[0.10] blur-[90px]"
                    style={{
                      background: 'radial-gradient(circle, hsl(199 89% 48%) 0%, transparent 70%)',
                      animation: 'aurora-3 26s ease-in-out infinite',
                    }}
                  />
                  {/* Noise texture overlay — grain SVG subtil pour l'effet premium.
                      baseFrequency 0.9 = grain fin, numOctaves 4 = texture riche. */}
                  <div
                    className="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]"
                    style={{
                      backgroundImage: `url("${noiseSvg}")`,
                      backgroundRepeat: 'repeat',
                      backgroundSize: '200px 200px',
                    }}
                  />
                </div>

                {/* Fond animé pluggable — la coquille gère defer/mobile/reduced-motion/cleanup
                    et rend le fond sélectionné (constellation, etc.). z-[1] : au-dessus des
                    blobs aurora (z-0), sous le contenu (z-10). */}
                <BackgroundHost />

                {/* Contenu au-dessus de l'aurora — z-10 garantit le passage devant les blobs */}
                <div className="relative z-10 flex min-h-screen flex-col">
                  {/* Tracker de clics (heatmap) — listener global, aucun rendu. Monté ici une fois pour
                      toutes les pages sous [locale]. */}
                  <ClickTracker />
                  <Header />
                  <main className="flex-1">{children}</main>
                  <Footer />
                </div>
              </div>
              </BackgroundProvider>
            </VisitorHeartbeatProvider>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
