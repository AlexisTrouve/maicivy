import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import Script from 'next/script';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { VisitorHeartbeatProvider } from '@/components/providers/VisitorHeartbeatProvider';
import { ThemeProvider } from '@/components/providers/ThemeProvider';
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
              <div className="relative flex min-h-screen flex-col bg-background">
                <Header />
                <main className="flex-1 bg-background">{children}</main>
                <Footer />
              </div>
            </VisitorHeartbeatProvider>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
