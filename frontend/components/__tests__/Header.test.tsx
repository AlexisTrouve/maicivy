import { fireEvent, render, screen, within } from '@testing-library/react';
import { usePathname } from '@/i18n/navigation';

import { Header } from '../layout/Header';

/**
 * Header — l'en-tête unique du site public (refonte montagne, 2026-11).
 *
 * Ce que ce test verrouille : la nav principale (même liste que le pied, lib/site/navigation.ts)
 * avec les bonnes cibles, l'entrée active signalée par aria-current, le menu téléphone (ouverture,
 * fermeture au clic, à Échap, nom accessible qui change), le contact en mailto, l'absence de
 * toute bascule de thème, et le rendu à plat (non collant) sur la route immersive (dossier vivant).
 */

// Navigation next-intl : Link rend un <a href> simple, usePathname est pilotable par test.
jest.mock('@/i18n/navigation', () => ({
  Link: ({ href, children, ...props }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  usePathname: jest.fn(() => '/'),
}));

// Le sélecteur de langue a son propre test (LanguageSwitcher.test.tsx).
jest.mock('@/components/shared/LanguageSwitcher', () => ({
  LanguageSwitcher: () => <div data-testid="lang-switcher" />,
}));

const mockedPathname = usePathname as jest.Mock;

/** Libellés FR attendus (messages/fr.json › nav), dans l'ordre de l'en-tête. */
const PRIMARY = [
  ['home', 'Accueil', '/'],
  ['dossierVivant', 'Dossier vivant', '/dossier-vivant'],
  ['portfolio', 'Portfolio', '/portfolio'],
  ['cv', 'CV', '/cv'],
  ['gitstats', 'Git Stats', '/gitstats'],
  ['blog', 'Blog', '/blog'],
  ['chat', 'Chat', '/chat'],
] as const;

describe('Header', () => {
  beforeEach(() => {
    mockedPathname.mockReturnValue('/');
  });

  it('le nom d’Alexis ramène à l’accueil (la montagne)', () => {
    render(<Header />);
    const home = screen.getByTestId('site-header-home');
    expect(home).toHaveTextContent('Alexis Trouvé');
    expect(home).toHaveAttribute('href', '/');
    expect(home).toHaveAccessibleName('Alexis Trouvé — accueil');
  });

  it('rend la nav principale, dans l’ordre, avec ses cibles', () => {
    render(<Header />);
    const nav = screen.getByRole('navigation', { name: 'Navigation principale' });
    const links = within(nav).getAllByRole('link');
    expect(links.map((a) => [a.textContent, a.getAttribute('href')])).toEqual(PRIMARY.map(([, label, href]) => [label, href]));
  });

  it('signale l’entrée active (aria-current + barre dorée), et elle seule', () => {
    mockedPathname.mockReturnValue('/blog/un-article');
    render(<Header />);
    const blog = screen.getByTestId('site-nav-blog');
    expect(blog).toHaveAttribute('aria-current', 'page');
    expect(blog.className).toContain('after:bg-gold');
    expect(screen.getByTestId('site-nav-home')).not.toHaveAttribute('aria-current');
  });

  it('l’accueil n’est actif que sur lui-même', () => {
    mockedPathname.mockReturnValue('/cv');
    render(<Header />);
    expect(screen.getByTestId('site-nav-home')).not.toHaveAttribute('aria-current');
    expect(screen.getByTestId('site-nav-cv')).toHaveAttribute('aria-current', 'page');
  });

  it('le contact est un mailto', () => {
    render(<Header />);
    expect(screen.getByTestId('site-header-contact')).toHaveAttribute('href', expect.stringMatching(/^mailto:.+@.+/));
  });

  it('menu téléphone : s’ouvre, liste les mêmes liens, se ferme au clic sur un lien', () => {
    render(<Header />);
    const toggle = screen.getByTestId('site-menu-toggle');
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(toggle).toHaveAccessibleName('Ouvrir le menu');
    expect(screen.queryByTestId('site-menu')).toBeNull();

    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    expect(toggle).toHaveAccessibleName('Fermer le menu');
    const menu = screen.getByTestId('site-menu');
    expect(within(menu).getAllByRole('link').map((a) => a.getAttribute('href'))).toEqual(PRIMARY.map(([, , href]) => href));

    fireEvent.click(screen.getByTestId('site-menu-portfolio'));
    expect(screen.queryByTestId('site-menu')).toBeNull();
  });

  it('menu téléphone : Échap le referme et rend le focus au bouton', () => {
    render(<Header />);
    const toggle = screen.getByTestId('site-menu-toggle');
    fireEvent.click(toggle);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByTestId('site-menu')).toBeNull();
    expect(toggle).toHaveFocus();
  });

  it('aucune bascule clair / sombre : le site est sombre en permanence', () => {
    render(<Header />);
    // Les seuls boutons de l'en-tête : le menu téléphone (le sélecteur de langue est bouchonné).
    expect(screen.getAllByRole('button')).toEqual([screen.getByTestId('site-menu-toggle')]);
    expect(screen.queryByLabelText(/thème|theme/i)).toBeNull();
  });

  it('accueil : ni collant ni empilé, pour que les calques plein écran de la montagne le couvrent', () => {
    const { rerender } = render(<Header />);
    const header = screen.getByTestId('site-header');
    // Ni sticky, ni z-index, ni backdrop-blur : chacun créerait un contexte au-dessus de la page.
    expect(header.className).not.toMatch(/\bsticky\b|\bz-\d|backdrop-blur/);
    // Le menu téléphone porte son propre z-index, pour rester au-dessus de la montagne.
    fireEvent.click(screen.getByTestId('site-menu-toggle'));
    expect(screen.getByTestId('site-menu').className).toContain('z-50');

    // Ailleurs : collant, au-dessus du contenu qui défile.
    mockedPathname.mockReturnValue('/portfolio');
    rerender(<Header />);
    expect(screen.getByTestId('site-header').className).toMatch(/\bsticky\b.*\bz-40\b/);
  });

  it('dossier vivant : présent, mais ni collant ni empilé — sa barre de rejeu colle dessous', () => {
    mockedPathname.mockReturnValue('/dossier-vivant');
    render(<Header />);
    const header = screen.getByTestId('site-header');
    expect(header.className).not.toMatch(/\bsticky\b|\bz-\d|backdrop-blur/);
    expect(screen.getByTestId('site-nav-dossierVivant')).toHaveAttribute('aria-current', 'page');
    // Téléphone couché : masqué (l'ouverture ne tiendrait pas sous deux barres) ; ailleurs, jamais.
    expect(header.className).toContain('[@media(orientation:landscape)_and_(max-height:500px)]:hidden');
    mockedPathname.mockReturnValue('/portfolio');
    render(<Header />);
    expect(screen.getAllByTestId('site-header')[1].className).not.toContain('max-height:500px');
  });
});
