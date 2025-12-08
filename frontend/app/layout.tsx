import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import './globals.css';

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

export const metadata: Metadata = {
  title: {
    default: 'maicivy - CV Interactif Intelligent',
    template: '%s | maicivy',
  },
  description: 'CV interactif avec génération de lettres de motivation par IA',
  keywords: ['CV', 'portfolio', 'IA', 'développeur', 'full-stack'],
  authors: [{ name: 'Alexi' }],
  openGraph: {
    type: 'website',
    locale: 'fr_FR',
    url: 'https://maicivy.com',
    title: 'maicivy - CV Interactif Intelligent',
    description: 'CV interactif avec génération de lettres de motivation par IA',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="fr" suppressHydrationWarning>
      <body className={`${inter.variable} ${poppins.variable} font-sans antialiased`}>
        <div className="relative flex min-h-screen flex-col">
          <Header />
          <main className="flex-1">{children}</main>
          <Footer />
        </div>
      </body>
    </html>
  );
}
