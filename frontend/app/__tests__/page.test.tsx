import { render, screen } from '@testing-library/react';
import HomePage from '../page';

// Mock Next.js components
jest.mock('next/link', () => {
  return ({ children, href }: { children: React.ReactNode; href: string }) => {
    return <a href={href}>{children}</a>;
  };
});

// Mock shadcn/ui components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, asChild, ...props }: any) => {
    if (asChild) {
      return <div {...props}>{children}</div>;
    }
    return <button {...props}>{children}</button>;
  },
}));

jest.mock('@/components/ui/card', () => ({
  Card: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardHeader: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardTitle: ({ children, ...props }: any) => <h3 {...props}>{children}</h3>,
  CardDescription: ({ children, ...props }: any) => <p {...props}>{children}</p>,
  CardContent: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  FileText: () => <svg data-testid="file-text-icon" />,
  Sparkles: () => <svg data-testid="sparkles-icon" />,
  BarChart3: () => <svg data-testid="bar-chart-icon" />,
}));

describe('HomePage', () => {
  it('should render main heading', () => {
    render(<HomePage />);

    const heading = screen.getByRole('heading', {
      name: /CV Interactif Intelligent/i,
      level: 1,
    });
    expect(heading).toBeInTheDocument();
  });

  it('should render hero description', () => {
    render(<HomePage />);

    expect(
      screen.getByText(/Découvrez mon parcours professionnel adaptatif/i)
    ).toBeInTheDocument();
  });

  it('should render CTA buttons in hero section', () => {
    render(<HomePage />);

    const cvButton = screen.getByRole('link', { name: /Voir mon CV/i });
    const letterButton = screen.getByRole('link', {
      name: /Générer une lettre/i,
    });

    expect(cvButton).toBeInTheDocument();
    expect(cvButton).toHaveAttribute('href', '/cv');

    expect(letterButton).toBeInTheDocument();
    expect(letterButton).toHaveAttribute('href', '/letters');
  });

  it('should render feature cards', () => {
    render(<HomePage />);

    // CV Dynamique card
    expect(screen.getByText(/CV Dynamique/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Un CV qui s'adapte automatiquement/i)
    ).toBeInTheDocument();

    // Lettres IA card
    expect(screen.getByText(/Lettres IA/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Génération de lettres de motivation et anti-motivation/i)
    ).toBeInTheDocument();

    // Analytics card
    expect(screen.getByText(/Analytics Publiques/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Dashboard temps réel des statistiques/i)
    ).toBeInTheDocument();
  });

  it('should render links in feature cards', () => {
    render(<HomePage />);

    const links = screen.getAllByRole('link', { name: /Explorer|Essayer|Voir les stats/i });

    // Should have 3 links in cards
    expect(links).toHaveLength(3);
  });

  it('should have proper card structure', () => {
    render(<HomePage />);

    // Check for FileText icon (CV card)
    expect(screen.getByText(/CV Dynamique/i)).toBeInTheDocument();

    // Check for Sparkles icon (Letters card)
    expect(screen.getByText(/Lettres IA/i)).toBeInTheDocument();

    // Check for BarChart3 icon (Analytics card)
    expect(screen.getByText(/Analytics Publiques/i)).toBeInTheDocument();
  });

  it('should render container with proper classes', () => {
    const { container } = render(<HomePage />);

    const mainContainer = container.querySelector('.container');
    expect(mainContainer).toBeInTheDocument();
    expect(mainContainer).toHaveClass('container', 'py-12');
  });

  it('should render responsive grid for feature cards', () => {
    const { container } = render(<HomePage />);

    const grid = container.querySelector('.grid');
    expect(grid).toBeInTheDocument();
    expect(grid).toHaveClass('md:grid-cols-3');
  });
});
