import { render, screen, within } from '@testing-library/react';

import { Footer } from '../layout/Footer';

/**
 * Footer — le pied de page unique du site public (refonte montagne, 2026-11).
 *
 * Verrouille : les quatre groupes de liens et leurs cibles (dont la nav principale — c'est par le
 * pied que le dossier vivant mène au reste du site), le contact en mailto, les profils externes
 * ouverts proprement, le sélecteur de langue ouvert vers le haut, et l'identité tirée de la voix
 * de la montagne (pas une copie).
 */

jest.mock('@/i18n/navigation', () => ({
  Link: ({ href, children, ...props }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

const switcherProps = jest.fn();
jest.mock('@/components/shared/LanguageSwitcher', () => ({
  LanguageSwitcher: (props: unknown) => {
    switcherProps(props);
    return <div data-testid="lang-switcher" />;
  },
}));

describe('Footer', () => {
  it('identité : nom, signature et phrase de la voix de la montagne', () => {
    render(<Footer />);
    const footer = screen.getByTestId('site-footer');
    expect(footer).toHaveTextContent('Alexis Trouvé');
    expect(footer).toHaveTextContent('Agent Orchestrator — Technomancer');
    expect(footer).toHaveTextContent("Je construis des systèmes d'agents IA qui tournent en production.");
  });

  it.each([
    ['home', '/'],
    ['dossierVivant', '/dossier-vivant'],
    ['portfolio', '/portfolio'],
    ['cv', '/cv'],
    ['blog', '/blog'],
    ['chat', '/chat'],
    ['gitstats', '/gitstats'],
    ['analytics', '/analytics'],
    ['architecture', '/architecture'],
    ['legalNotice', '/legal'],
    ['privacy', '/privacy'],
  ])('lien %s → %s', (id, href) => {
    render(<Footer />);
    expect(screen.getByTestId(`site-footer-${id}`)).toHaveAttribute('href', href);
  });

  it('libellés des coulisses et du légal (i18n FR)', () => {
    render(<Footer />);
    expect(screen.getByTestId('site-footer-gitstats')).toHaveTextContent('Git Stats');
    // L'atelier de lettres public a été retiré (09/11/2026) : plus de lien dans les coulisses.
    expect(screen.queryByTestId('site-footer-letters')).toBeNull();
    expect(screen.getByTestId('site-footer-legalNotice')).toHaveTextContent('Mentions légales');
    expect(screen.getByRole('heading', { name: 'Coulisses' })).toBeInTheDocument();
  });

  it('contact : mailto, et profils externes dans un nouvel onglet sans opener', () => {
    render(<Footer />);
    expect(screen.getByTestId('site-footer-email')).toHaveAttribute('href', 'mailto:alexistrouve.pro@gmail.com');
    for (const id of ['linkedin', 'github']) {
      const a = screen.getByTestId(`site-footer-${id}`);
      expect(a).toHaveAttribute('target', '_blank');
      expect(a).toHaveAttribute('rel', 'noopener noreferrer');
      expect(a.getAttribute('href')).toMatch(/^https:\/\//);
    }
  });

  it('droits avec l’année courante, et le sélecteur de langue ouvert vers le haut', () => {
    render(<Footer />);
    expect(screen.getByText(`© ${new Date().getFullYear()} Alexis Trouvé. Tous droits réservés.`)).toBeInTheDocument();
    expect(within(screen.getByTestId('site-footer-language')).getByTestId('lang-switcher')).toBeInTheDocument();
    expect(switcherProps).toHaveBeenCalledWith({ placement: 'up' });
  });
});
