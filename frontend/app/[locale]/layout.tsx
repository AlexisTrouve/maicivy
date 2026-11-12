import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { NextIntlClientProvider, hasLocale } from 'next-intl';
import { getMessages } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { VisitorHeartbeatProvider } from '@/components/providers/VisitorHeartbeatProvider';
import ClickTracker from '@/components/analytics/ClickTracker';
import RefTracker from '@/components/analytics/RefTracker';
import { locales } from '@/i18n/config';
import { loadMessages } from '@/i18n/messages';
import { montagneSerif } from '@/lib/montagne/fonts';
import { siteBaseUrl, socialMetadata } from '@/lib/seo/social';
import montagneStyles from '@/components/montagne/montagne.module.css';
import '../globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

// Generate dynamic metadata based on locale
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const resolvedParams = await params;
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

  const baseUrl = siteBaseUrl();
  // Aperçu par défaut des pages sans métadonnées propres. Les pages qui ont leur titre passent par
  // le même helper : redéclarer `openGraph` sans image effacerait celle-ci (fusion clé par clé).
  const social = socialMetadata({ locale, title: messages.metadata.title, description: messages.metadata.description });

  return {
    title: {
      default: messages.metadata.title,
      template: '%s | maicivy',
    },
    description: messages.metadata.description,
    keywords: ['Alexis Trouvé', 'agent orchestrator', 'technomancer', 'AI agents', 'agents IA', 'multi-agent', 'portfolio'],
    authors: [{ name: 'Alexis Trouvé' }],
    creator: 'Alexis Trouvé',
    metadataBase: new URL(baseUrl),
    ...social,
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
  params: Promise<{ locale: string }>;
}) {
  const resolvedParams = await params;
  const locale = resolvedParams.locale;
  // Voir generateMetadata pour le QUOI/POURQUOI/COMMENT. Le layout est le point DRY qui enveloppe
  // toutes les pages sous [locale] → valider ici protège chaque route. notFound() AVANT getMessages
  // (sinon on chargerait des messages pour une locale qui n'existe pas).
  if (!hasLocale(locales, locale)) {
    notFound();
  }
  const messages = await getMessages({ locale });

  // QUOI : retire le namespace `admin` des messages envoyés au client SUR LES PAGES PUBLIQUES.
  // POURQUOI : NextIntlClientProvider sérialise l'objet reçu dans le HTML de CHAQUE page. Passer
  //   `messages` entier publiait donc les libellés du panneau owner (titres des sept outils, aides
  //   de saisie, messages d'erreur d'auth) sur maicivy.etheryale.com, lisibles dans un view-source
  //   ou par n'importe quel crawler. Ce n'est pas une fuite de données, mais une fuite de PRODUIT :
  //   ces libellés décrivent l'outil interne. Le namespace pèse ~20 % du fichier de messages et
  //   AUCUN composant sous app/[locale] ou components ne le lit (vérifié par grep) : il était
  //   expédié pour rien.
  // COMMENT : destructuration qui écarte `admin` et garde le reste. `app/admin/layout.tsx` continue
  //   de recevoir l'objet complet — il est hors de l'arbre [locale] et derrière l'auth owner.
  //   ⚠️ Si un composant public a un jour besoin d'une clé de `admin`, on la déplace dans `common`,
  //   on ne réintroduit pas le namespace ici. tests/e2e/public-no-admin-messages.spec.ts verrouille.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars -- `admin` est extrait pour être ÉCARTÉ.
  const { admin: _adminMessages, ...publicMessages } = messages as Record<string, unknown>;

  return (
    // `dark` EN DUR : le site est sombre en permanence, il n'y a plus de bascule ni de script
    // anti-flash. La classe ne choisit plus de palette (globals.css n'en a qu'une) ; elle garde
    // actives les variantes `dark:` des pages internes, écrites du temps des deux thèmes.
    <html lang={locale} className="dark">
      {/* site-public (globals.css) : titres h1–h3 dans la serif de la montagne. Inter reste la
          police du texte courant. */}
      <body className={`${inter.variable} ${montagneSerif.variable} site-public font-sans antialiased bg-background text-foreground`}>
        <NextIntlClientProvider messages={publicMessages}>
          <VisitorHeartbeatProvider showActiveVisitors={false}>
            <div className="relative isolate flex min-h-screen flex-col">
              {/* Le ciel de la montagne (montagne.module.css › .sky, même définition que l'accueil)
                  fixé derrière toutes les pages : il remplace les taches floues et le fond animé.
                  Statique — aucun calcul hors champ. */}
              <div aria-hidden="true" className={`${montagneStyles.sky} pointer-events-none fixed inset-0 -z-10`} />
              {/* Tracker de clics (heatmap) — listener global, aucun rendu. Monté ici une fois pour
                  toutes les pages sous [locale]. */}
              <ClickTracker />
              <RefTracker />
              <Header />
              <main className="flex-1">{children}</main>
              <Footer />
            </div>
          </VisitorHeartbeatProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
