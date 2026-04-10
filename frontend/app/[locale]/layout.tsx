import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import Script from 'next/script';
import nextDynamic from 'next/dynamic';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { VisitorHeartbeatProvider } from '@/components/providers/VisitorHeartbeatProvider';
import { ThemeProvider } from '@/components/providers/ThemeProvider';
import '../globals.css';

// Import lazy — ssr:false garantit 0 exécution server-side.
// Three.js est lui-même importé dynamiquement à l'intérieur du composant (useEffect),
// ce qui donne deux couches de lazy loading : Next.js chunking + import() natif.
const ConstellationBackground = nextDynamic(
  () => import('@/components/background/ConstellationBackground'),
  { ssr: false }
);

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

// Force dynamic rendering for all locale pages to ensure proper i18n context
export const dynamic = 'force-dynamic';

// Generate dynamic metadata based on locale
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }> | { locale: string };
}): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale || 'fr';
  const messages = (await import(`@/messages/${locale}.json`)).default;

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
          // Image statique hébergée sur maiprofiles (l'endpoint /api/og est routé vers le backend Go, pas Next.js)
          url: 'https://maiprofiles.etheryale.com/images/img_dbb0624c',
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
      images: ['https://maiprofiles.etheryale.com/images/img_dbb0624c'],
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

                {/* Constellation 3D Three.js — lazy loaded après hydration.
                    z-[1] : au-dessus des blobs aurora (z-0), sous le contenu (z-10).
                    Le composant gère lui-même son canvas en position fixed. */}
                <ConstellationBackground />

                {/* Contenu au-dessus de l'aurora — z-10 garantit le passage devant les blobs */}
                <div className="relative z-10 flex min-h-screen flex-col">
                  <Header />
                  <main className="flex-1">{children}</main>
                  <Footer />
                </div>
              </div>
            </VisitorHeartbeatProvider>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
