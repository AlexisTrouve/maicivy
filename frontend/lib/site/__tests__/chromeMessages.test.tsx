import { fireEvent, render, screen } from '@testing-library/react';

import fr from '@/messages/fr.json';
import en from '@/messages/en.json';
import de from '@/messages/de.json';
import itMsgs from '@/messages/it.json';
import zh from '@/messages/zh.json';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { LanguageSwitcher } from '@/components/shared/LanguageSwitcher';

/**
 * Chrome du site — chaque texte de l'en-tête, du menu téléphone, du pied de page et du sélecteur de
 * langue existe dans les CINQ langues, et il y est TRADUIT.
 *
 * POURQUOI : la chrome est sur toutes les pages ; une clé absente s'y affiche brute partout, et une
 * valeur restée en français y met en échec chaque visiteur de/it/zh dès le premier écran.
 * COMMENT : les clés ne sont PAS recopiées ici dans une liste qui pourrait vieillir — on rend les
 * vrais composants (menu ouvert, sélecteur déplié) en ENREGISTRANT chaque clé que leurs `t()`
 * demandent, puis on vérifie chacune dans les cinq fichiers. Une clé ajoutée demain à l'en-tête
 * entre d'elle-même dans le test.
 */

const requested = new Set<string>();

// Enveloppe le mock global de next-intl (qui résout le FR) pour noter chaque clé demandée.
jest.mock('next-intl', () => {
  const actual = jest.requireActual('next-intl');
  return {
    ...actual,
    useTranslations: (namespace?: string) => {
      const t = actual.useTranslations(namespace);
      const recording = (key: string, values?: Record<string, unknown>) => {
        requested.add(namespace ? `${namespace}.${key}` : key);
        return t(key, values);
      };
      recording.raw = t.raw;
      return recording;
    },
  };
});

jest.mock('@/i18n/navigation', () => ({
  Link: ({ href, children, ...props }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  usePathname: () => '/',
  useRouter: () => ({ replace: () => {} }),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ locale: 'fr' }),
  usePathname: () => '/fr',
}));

type Dict = Record<string, unknown>;
const LOCALES: Record<string, Dict> = { fr, en, de, it: itMsgs, zh };

function lookup(dict: Dict, path: string): unknown {
  return path.split('.').reduce<unknown>((o, k) => (o && typeof o === 'object' ? (o as Dict)[k] : undefined), dict);
}

/**
 * Valeurs légitimement identiques au français dans une autre langue : noms propres, marques, la
 * signature gardée en anglais partout (décision de la maquette AC-2), et les mots que la langue
 * cible écrit pareil (« Contact » et « Architecture » en anglais, « Portfolio », « Blog », « Chat », « CV »).
 */
const SAME_AS_FR = new Set([
  'nav.brand',
  'nav.portfolio',
  'nav.blog',
  'nav.chat',
  'nav.gitstats',
  'nav.analytics',
  'nav.cv',
  'nav.architecture',
  'nav.contact',
  'footer.contact',
  'footer.linkedin',
  'footer.github',
  'montagne.voice.tagline',
]);

beforeAll(() => {
  render(
    <>
      <Header />
      <Footer />
      <LanguageSwitcher />
    </>,
  );
  // Tout ce qui n'est rendu qu'à l'ouverture : le menu téléphone et la liste des langues.
  fireEvent.click(screen.getByTestId('site-menu-toggle'));
  for (const b of screen.getAllByRole('button', { expanded: false })) fireEvent.click(b);
});

describe('chrome du site — messages dans les 5 langues', () => {
  it('a bien relevé les clés de l’en-tête, du menu, du pied et du sélecteur', () => {
    // Garde-fou du garde-fou : si l'enregistrement cassait, le test passerait à vide.
    for (const k of ['nav.openMenu', 'nav.closeMenu', 'nav.dossierVivant', 'footer.workshop', 'footer.rights', 'nav.language', 'nav.languageList']) {
      expect(requested).toContain(k);
    }
  });

  it.each(Object.keys(LOCALES))('%s : chaque clé existe et n’est pas vide', (locale) => {
    const missing = [...requested].filter((k) => {
      const v = lookup(LOCALES[locale], k);
      return typeof v !== 'string' || v.trim() === '';
    });
    expect(missing).toEqual([]);
  });

  it.each(['en', 'de', 'it', 'zh'])('%s : traduit, pas recopié du français', (locale) => {
    const copied = [...requested].filter((k) => !SAME_AS_FR.has(k) && lookup(LOCALES[locale], k) === lookup(fr, k));
    expect(copied).toEqual([]);
  });
});
