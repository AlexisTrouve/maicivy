import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
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
  description: 'CV interactif avec generation de lettres de motivation par IA',
  keywords: ['CV', 'portfolio', 'IA', 'developpeur', 'full-stack'],
  authors: [{ name: 'Alexis' }],
  openGraph: {
    type: 'website',
    locale: 'fr_FR',
    url: 'https://maicivy.com',
    title: 'maicivy - CV Interactif Intelligent',
    description: 'CV interactif avec generation de lettres de motivation par IA',
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
        {children}
      </body>
    </html>
  );
}
