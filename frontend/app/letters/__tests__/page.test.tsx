import { render, screen } from '@testing-library/react';
import LettersPage, { metadata } from '../page';

// Mock letter components
jest.mock('@/components/letters/LetterGenerator', () => {
  return {
    LetterGenerator: function MockLetterGenerator() {
      return (
        <div data-testid="letter-generator">
          <form>
            <input name="companyName" placeholder="Company Name" />
            <input name="jobTitle" placeholder="Job Title" />
            <button type="submit">Generate</button>
          </form>
        </div>
      );
    },
  };
});

jest.mock('@/components/letters/AccessGate', () => {
  return {
    AccessGate: function MockAccessGate({
      children,
    }: {
      children: React.ReactNode;
    }) {
      return <div data-testid="access-gate">{children}</div>;
    },
  };
});

describe('LettersPage', () => {
  it('should render main heading', () => {
    render(<LettersPage />);

    expect(
      screen.getByRole('heading', { name: /Générateur de Lettres par IA/i })
    ).toBeInTheDocument();
  });

  it('should render description text', () => {
    render(<LettersPage />);

    expect(
      screen.getByText(/Générez instantanément une lettre de motivation/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/version humoristique "anti-motivation"/i)
    ).toBeInTheDocument();
  });

  it('should render AccessGate component', () => {
    render(<LettersPage />);

    const accessGate = screen.getByTestId('access-gate');
    expect(accessGate).toBeInTheDocument();
  });

  it('should render LetterGenerator inside AccessGate', () => {
    render(<LettersPage />);

    const letterGenerator = screen.getByTestId('letter-generator');
    expect(letterGenerator).toBeInTheDocument();

    // Verify it's inside AccessGate
    const accessGate = screen.getByTestId('access-gate');
    expect(accessGate).toContainElement(letterGenerator);
  });

  it('should have gradient background classes', () => {
    const { container } = render(<LettersPage />);

    const mainDiv = container.querySelector('.min-h-screen');
    expect(mainDiv).toBeInTheDocument();
    expect(mainDiv).toHaveClass('bg-gradient-to-br');
  });

  it('should render container with proper classes', () => {
    const { container } = render(<LettersPage />);

    const containerDiv = container.querySelector('.container');
    expect(containerDiv).toBeInTheDocument();
    expect(containerDiv).toHaveClass('mx-auto', 'px-4', 'py-12');
  });

  it('should render heading with gradient text', () => {
    const { container } = render(<LettersPage />);

    const heading = screen.getByRole('heading', {
      name: /Générateur de Lettres par IA/i,
    });
    expect(heading).toHaveClass('bg-clip-text', 'text-transparent');
  });

  it('should have responsive text sizing', () => {
    render(<LettersPage />);

    const heading = screen.getByRole('heading', {
      name: /Générateur de Lettres par IA/i,
    });
    expect(heading).toHaveClass('text-4xl', 'md:text-5xl');
  });

  it('should render centered header section', () => {
    const { container } = render(<LettersPage />);

    const headerDiv = container.querySelector('.text-center.mb-12');
    expect(headerDiv).toBeInTheDocument();
  });

  it('should render description with proper styling', () => {
    const { container } = render(<LettersPage />);

    const description = screen.getByText(/Générez instantanément/i);
    expect(description).toHaveClass('text-lg');
  });
});

describe('LettersPage - Metadata', () => {
  it('should have correct title', () => {
    expect(metadata.title).toBe('Générateur de Lettres IA | maicivy');
  });

  it('should have correct description', () => {
    expect(metadata.description).toContain(
      'Générez des lettres de motivation et anti-motivation'
    );
    expect(metadata.description).toContain('personnalisées par IA');
  });

  it('should have OpenGraph metadata', () => {
    expect(metadata.openGraph).toBeDefined();
    expect(metadata.openGraph?.title).toBe('Générateur de Lettres IA');
    expect(metadata.openGraph?.description).toContain(
      'Lettres de motivation/anti-motivation'
    );
  });
});
