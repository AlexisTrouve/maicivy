import { render, screen } from '@testing-library/react';
import { Header } from '../layout/Header';
import { usePathname } from '@/i18n/navigation';

// --- Mocks des dépendances feuilles du VRAI Header ---
// (avant : ce fichier testait un Header mock inline sans rapport avec le composant réel → garde-fou nul)

// Navigation next-intl : Link rend un <a href> simple, usePathname est pilotable par test.
jest.mock('@/i18n/navigation', () => ({
  Link: ({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) => (
    <a href={href} className={className}>{children}</a>
  ),
  usePathname: jest.fn(() => '/'),
}));

// Thème : on neutralise le hook (le toggle n'est pas l'objet du test des onglets).
jest.mock('@/hooks/useTheme', () => ({
  useTheme: () => ({ theme: 'light', toggleTheme: jest.fn() }),
}));

// Switchers latéraux : stubs — leurs deps internes ne concernent pas la nav.
jest.mock('@/components/shared/LanguageSwitcher', () => ({
  LanguageSwitcher: () => <div data-testid="lang-switcher" />,
}));
jest.mock('@/components/background/BackgroundSwitcher', () => ({
  BackgroundSwitcher: () => <div data-testid="bg-switcher" />,
}));

// next-intl (useTranslations) + lucide sont fournis par les mocks GLOBAUX (jest.config moduleNameMapper).
// useTranslations('nav') résout les vraies clés FR : home→Accueil, cv→CV, etc.

const mockedPathname = usePathname as jest.Mock;

describe('Header', () => {
  beforeEach(() => {
    mockedPathname.mockReturnValue('/');
  });

  it('rend le logo qui pointe vers la home', () => {
    render(<Header />);
    const logo = screen.getByText('maicivy');
    expect(logo.closest('a')).toHaveAttribute('href', '/');
  });

  it('rend tous les onglets de navigation (i18n FR)', () => {
    render(<Header />);
    ['Accueil', 'CV', 'Lettres', 'Chat', 'Analytics', 'Blog', 'Git Stats'].forEach((label) => {
      expect(screen.getByText(label)).toBeInTheDocument();
    });
  });

  it('rend chaque onglet en pastille (rounded-full + padding)', () => {
    render(<Header />);
    const cv = screen.getByText('CV');
    expect(cv).toHaveClass('rounded-full', 'px-3');
  });

  it("met l'onglet actif en pastille PLEINE (bg-primary + texte contrasté)", () => {
    mockedPathname.mockReturnValue('/cv');
    render(<Header />);
    const cv = screen.getByText('CV');
    expect(cv).toHaveClass('bg-primary', 'text-primary-foreground');
  });

  it("laisse les onglets inactifs en texte (pas de fond plein)", () => {
    mockedPathname.mockReturnValue('/cv');
    render(<Header />);
    const home = screen.getByText('Accueil');
    expect(home).toHaveClass('text-muted-foreground');
    expect(home).not.toHaveClass('bg-primary');
  });

  it('chaque onglet a un href correct', () => {
    render(<Header />);
    expect(screen.getByText('CV').closest('a')).toHaveAttribute('href', '/cv');
    expect(screen.getByText('Lettres').closest('a')).toHaveAttribute('href', '/letters');
    expect(screen.getByText('Git Stats').closest('a')).toHaveAttribute('href', '/gitstats');
  });
});
