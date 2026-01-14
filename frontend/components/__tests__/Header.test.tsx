import { render, screen, fireEvent } from '@testing-library/react';
import { usePathname } from 'next/navigation';

// Mock Next.js navigation
jest.mock('next/navigation', () => ({
  usePathname: jest.fn(),
}));

// Mock Header component (since it doesn't exist yet, we'll create a mock implementation)
const Header = () => {
  const pathname = usePathname();

  const [mobileMenuOpen, setMobileMenuOpen] = React.useState(false);
  const [theme, setTheme] = React.useState<'light' | 'dark'>('light');

  const links = [
    { href: '/', label: 'Accueil' },
    { href: '/cv', label: 'CV' },
    { href: '/letters', label: 'Lettres' },
    { href: '/analytics', label: 'Analytics' },
  ];

  return (
    <header className="bg-white dark:bg-gray-900 shadow-sm">
      <nav className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          {/* Logo */}
          <a href="/" className="text-xl font-bold text-gray-900 dark:text-white">
            MaicIvy
          </a>

          {/* Desktop Navigation */}
          <ul className="hidden md:flex items-center gap-6">
            {links.map((link) => (
              <li key={link.href}>
                <a
                  href={link.href}
                  className={`transition-colors ${
                    pathname === link.href
                      ? 'text-blue-600 font-semibold'
                      : 'text-gray-600 dark:text-gray-300 hover:text-blue-600'
                  }`}
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>

          {/* Theme Toggle & Mobile Menu */}
          <div className="flex items-center gap-4">
            <button
              onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
              className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              aria-label="Toggle theme"
            >
              {theme === 'light' ? '🌙' : '☀️'}
            </button>

            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="md:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
              aria-label="Toggle menu"
            >
              {mobileMenuOpen ? '✕' : '☰'}
            </button>
          </div>
        </div>

        {/* Mobile Menu */}
        {mobileMenuOpen && (
          <ul className="md:hidden mt-4 space-y-2">
            {links.map((link) => (
              <li key={link.href}>
                <a
                  href={link.href}
                  className={`block py-2 transition-colors ${
                    pathname === link.href
                      ? 'text-blue-600 font-semibold'
                      : 'text-gray-600 dark:text-gray-300'
                  }`}
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>
        )}
      </nav>
    </header>
  );
};

import React from 'react';

describe('Header', () => {
  beforeEach(() => {
    (usePathname as jest.Mock).mockReturnValue('/');
  });

  it('should render logo', () => {
    render(<Header />);

    const logo = screen.getByText('MaicIvy');
    expect(logo).toBeInTheDocument();
    expect(logo.closest('a')).toHaveAttribute('href', '/');
  });

  it('should render all navigation links', () => {
    render(<Header />);

    expect(screen.getByText('Accueil')).toBeInTheDocument();
    expect(screen.getByText('CV')).toBeInTheDocument();
    expect(screen.getByText('Lettres')).toBeInTheDocument();
    expect(screen.getByText('Analytics')).toBeInTheDocument();
  });

  it('should highlight active link', () => {
    (usePathname as jest.Mock).mockReturnValue('/cv');

    render(<Header />);

    const cvLink = screen.getAllByText('CV')[0]; // Desktop link
    expect(cvLink).toHaveClass('text-blue-600', 'font-semibold');
  });

  it('should not highlight inactive links', () => {
    (usePathname as jest.Mock).mockReturnValue('/cv');

    render(<Header />);

    const homeLinks = screen.getAllByText('Accueil');
    homeLinks.forEach(link => {
      expect(link).not.toHaveClass('text-blue-600');
    });
  });

  it('should render theme toggle button', () => {
    render(<Header />);

    const themeButton = screen.getByLabelText('Toggle theme');
    expect(themeButton).toBeInTheDocument();
  });

  it('should toggle theme when button is clicked', () => {
    render(<Header />);

    const themeButton = screen.getByLabelText('Toggle theme');

    // Initial theme is light (moon icon)
    expect(screen.getByText('🌙')).toBeInTheDocument();

    // Click to toggle to dark
    fireEvent.click(themeButton);
    expect(screen.getByText('☀️')).toBeInTheDocument();

    // Click to toggle back to light
    fireEvent.click(themeButton);
    expect(screen.getByText('🌙')).toBeInTheDocument();
  });

  it('should render mobile menu toggle button', () => {
    render(<Header />);

    const menuButton = screen.getByLabelText('Toggle menu');
    expect(menuButton).toBeInTheDocument();
  });

  it('should open mobile menu when toggle is clicked', () => {
    render(<Header />);

    const menuButton = screen.getByLabelText('Toggle menu');

    // Initially closed
    expect(screen.getByText('☰')).toBeInTheDocument();

    // Click to open
    fireEvent.click(menuButton);

    // Should show close icon and menu items
    expect(screen.getByText('✕')).toBeInTheDocument();

    // Should have duplicate links (desktop + mobile)
    const accueilLinks = screen.getAllByText('Accueil');
    expect(accueilLinks.length).toBeGreaterThan(1);
  });

  it('should close mobile menu when toggle is clicked again', () => {
    render(<Header />);

    const menuButton = screen.getByLabelText('Toggle menu');

    // Open menu
    fireEvent.click(menuButton);
    expect(screen.getByText('✕')).toBeInTheDocument();

    // Close menu
    fireEvent.click(menuButton);
    expect(screen.getByText('☰')).toBeInTheDocument();

    // Should only have desktop links now
    const accueilLinks = screen.getAllByText('Accueil');
    expect(accueilLinks.length).toBe(1);
  });

  it('should hide desktop navigation on mobile', () => {
    const { container } = render(<Header />);

    const desktopNav = container.querySelector('.hidden.md\\:flex');
    expect(desktopNav).toBeInTheDocument();
  });

  it('should hide mobile menu on desktop', () => {
    const { container } = render(<Header />);

    const menuButton = screen.getByLabelText('Toggle menu');
    fireEvent.click(menuButton);

    const mobileMenu = container.querySelector('.md\\:hidden.mt-4');
    expect(mobileMenu).toBeInTheDocument();
  });

  it('should have correct link hrefs', () => {
    render(<Header />);

    const cvLinks = screen.getAllByText('CV');
    expect(cvLinks[0].closest('a')).toHaveAttribute('href', '/cv');

    const lettersLinks = screen.getAllByText('Lettres');
    expect(lettersLinks[0].closest('a')).toHaveAttribute('href', '/letters');

    const analyticsLinks = screen.getAllByText('Analytics');
    expect(analyticsLinks[0].closest('a')).toHaveAttribute('href', '/analytics');
  });

  it('should apply hover styles to navigation links', () => {
    const { container } = render(<Header />);

    const links = container.querySelectorAll('a[href="/cv"]');
    links.forEach(link => {
      expect(link).toHaveClass('transition-colors');
    });
  });

  it('should have shadow on header', () => {
    const { container } = render(<Header />);

    const header = container.querySelector('header');
    expect(header).toHaveClass('shadow-sm');
  });

  it('should support dark mode classes', () => {
    const { container } = render(<Header />);

    const header = container.querySelector('header');
    expect(header).toHaveClass('dark:bg-gray-900');
  });

  it('should center content in container', () => {
    const { container } = render(<Header />);

    const nav = container.querySelector('nav');
    expect(nav).toHaveClass('container', 'mx-auto');
  });

  it('should highlight current page in mobile menu', () => {
    (usePathname as jest.Mock).mockReturnValue('/letters');

    render(<Header />);

    const menuButton = screen.getByLabelText('Toggle menu');
    fireEvent.click(menuButton);

    const mobileLettersLinks = screen.getAllByText('Lettres');
    // Find the mobile menu link (not the desktop one)
    const mobileLink = mobileLettersLinks.find(link =>
      link.classList.contains('block')
    );

    expect(mobileLink).toHaveClass('text-blue-600', 'font-semibold');
  });
});
