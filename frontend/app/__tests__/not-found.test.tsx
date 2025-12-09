import { render, screen } from '@testing-library/react';
import NotFound from '../not-found';

// Mock Next.js components
jest.mock('next/link', () => {
  return ({ children, href }: { children: React.ReactNode; href: string }) => {
    return <a href={href}>{children}</a>;
  };
});

// Mock UI components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, asChild, className }: any) => {
    if (asChild) {
      return <div className={className}>{children}</div>;
    }
    return <button className={className}>{children}</button>;
  },
}));

describe('NotFound Page', () => {
  it('should render 404 heading', () => {
    render(<NotFound />);

    const heading = screen.getByRole('heading', { name: /404/i });
    expect(heading).toBeInTheDocument();
  });

  it('should render 404 with large font', () => {
    render(<NotFound />);

    const heading = screen.getByRole('heading', { name: /404/i });
    expect(heading).toHaveClass('text-6xl', 'font-bold');
  });

  it('should render "Page non trouvée" text', () => {
    render(<NotFound />);

    expect(screen.getByText(/Page non trouvée/i)).toBeInTheDocument();
  });

  it('should render description with proper styling', () => {
    render(<NotFound />);

    const description = screen.getByText(/Page non trouvée/i);
    expect(description).toHaveClass('text-xl', 'text-muted-foreground');
  });

  it('should render back to home link', () => {
    render(<NotFound />);

    const link = screen.getByRole('link', { name: /Retour à l'accueil/i });
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute('href', '/');
  });

  it('should have minimum height of screen', () => {
    const { container } = render(<NotFound />);

    const mainDiv = container.querySelector('.min-h-screen');
    expect(mainDiv).toBeInTheDocument();
  });

  it('should center content vertically and horizontally', () => {
    const { container } = render(<NotFound />);

    const mainDiv = container.querySelector('.container');
    expect(mainDiv).toHaveClass(
      'flex',
      'min-h-screen',
      'flex-col',
      'items-center',
      'justify-center'
    );
  });

  it('should have proper spacing between elements', () => {
    render(<NotFound />);

    const description = screen.getByText(/Page non trouvée/i);
    expect(description).toHaveClass('mt-4');
  });

  it('should have proper button spacing', () => {
    const { container } = render(<NotFound />);

    const buttonContainer = container.querySelector('.mt-8');
    expect(buttonContainer).toBeInTheDocument();
  });

  it('should render elements in correct order', () => {
    const { container } = render(<NotFound />);

    const mainContainer = container.querySelector('.container');
    const children = mainContainer?.children;

    // First child should be 404 heading
    expect(children?.[0]).toHaveTextContent('404');

    // Second child should be description
    expect(children?.[1]).toHaveTextContent('Page non trouvée');

    // Third child should be button/link
    expect(children?.[2]).toContainElement(
      screen.getByRole('link', { name: /Retour à l'accueil/i })
    );
  });

  it('should have container class', () => {
    const { container } = render(<NotFound />);

    const mainDiv = container.querySelector('.container');
    expect(mainDiv).toBeInTheDocument();
  });

  it('should render flexbox column layout', () => {
    const { container } = render(<NotFound />);

    const mainDiv = container.querySelector('.container');
    expect(mainDiv).toHaveClass('flex-col');
  });

  it('should have accessible heading structure', () => {
    render(<NotFound />);

    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('404');
  });

  it('should have descriptive link text for accessibility', () => {
    render(<NotFound />);

    const link = screen.getByRole('link');
    expect(link).toHaveAccessibleName(/Retour à l'accueil/i);
  });
});
