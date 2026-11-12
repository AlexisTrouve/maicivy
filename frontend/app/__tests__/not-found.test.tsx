import { render, screen } from '@testing-library/react';
import NotFound from '../not-found';

// Mock Next.js components
jest.mock('next/link', () => {
  return ({ children, href, className }: { children: React.ReactNode; href: string; className?: string }) => {
    return <a href={href} className={className}>{children}</a>;
  };
});

// Mock UI components (kept in case other tests in this suite use it)
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

    // Component renders "Page non trouvee / Page not found" (no accent, bilingual)
    expect(screen.getByText(/Page non trouvee/i)).toBeInTheDocument();
  });

  it('should render description with proper styling', () => {
    render(<NotFound />);

    // Component renders "Page non trouvee / Page not found"
    const description = screen.getByText(/Page non trouvee/i);
    expect(description).toHaveClass('text-xl', 'text-muted-foreground');
  });

  it('should render back to home link', () => {
    render(<NotFound />);

    // Component renders "Retour / Back" as the link text
    const link = screen.getByRole('link', { name: /Retour/i });
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

    // The main div uses flex min-h-screen flex-col items-center justify-center (no .container class)
    const mainDiv = container.querySelector('.flex.min-h-screen');
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

    // The <p> description has mt-4 class
    const description = screen.getByText(/Page non trouvee/i);
    expect(description).toHaveClass('mt-4');
  });

  it('should have proper button spacing', () => {
    const { container } = render(<NotFound />);

    // The link itself carries the mt-8 class (no separate wrapper element)
    const linkWithSpacing = container.querySelector('.mt-8');
    expect(linkWithSpacing).toBeInTheDocument();
  });

  it('should render elements in correct order', () => {
    const { container } = render(<NotFound />);

    // The main container is the root flex div (no .container class)
    const mainContainer = container.querySelector('.flex.min-h-screen');
    const children = mainContainer?.children;

    // First child should be 404 heading
    expect(children?.[0]).toHaveTextContent('404');

    // Second child should be description
    expect(children?.[1]).toHaveTextContent('Page non trouvee');

    // Third child should be the back link
    expect(children?.[2]).toContainElement(
      screen.getByRole('link', { name: /Retour/i })
    );
  });

  it('should have container class', () => {
    const { container } = render(<NotFound />);

    // The main wrapper uses flex + min-h-screen (no separate .container class)
    const mainDiv = container.querySelector('.flex.min-h-screen');
    expect(mainDiv).toBeInTheDocument();
  });

  it('should render flexbox column layout', () => {
    const { container } = render(<NotFound />);

    // The main wrapper has flex-col
    const mainDiv = container.querySelector('.flex-col');
    expect(mainDiv).toHaveClass('flex-col');
  });

  it('should have accessible heading structure', () => {
    render(<NotFound />);

    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('404');
  });

  it('should have descriptive link text for accessibility', () => {
    render(<NotFound />);

    // Component renders "Retour / Back" as the link accessible name
    const link = screen.getByRole('link');
    expect(link).toHaveAccessibleName(/Retour/i);
  });
});
