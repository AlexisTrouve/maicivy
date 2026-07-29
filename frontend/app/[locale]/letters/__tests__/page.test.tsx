import { render, screen } from '@testing-library/react';
import LettersPage, { generateMetadata } from '../page';

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

jest.mock('@/components/letters/MessageGenerator', () => {
  return {
    MessageGenerator: function MockMessageGenerator() {
      return <div data-testid="message-generator">Message Generator</div>;
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

// LettersTabs reçoit letterGenerator et messageGenerator comme props et les rend
jest.mock('@/components/letters/LettersTabs', () => {
  return {
    LettersTabs: function MockLettersTabs({
      letterGenerator,
      messageGenerator,
    }: {
      letterGenerator: React.ReactNode;
      messageGenerator: React.ReactNode;
    }) {
      return (
        <div data-testid="letters-tabs">
          {letterGenerator}
          {messageGenerator}
        </div>
      );
    },
  };
});

// Paramètres par défaut pour le composant serveur async
const defaultParams = { params: { locale: 'fr' } };

describe('LettersPage', () => {
  it('should render main heading', async () => {
    render(await LettersPage(defaultParams));

    // t('title') = "Générateur de Lettres par IA"
    expect(
      screen.getByRole('heading', { name: /Générateur de Lettres par IA/i })
    ).toBeInTheDocument();
  });

  it('should render description text', async () => {
    render(await LettersPage(defaultParams));

    // t('subtitle') = "Générez instantanément une lettre de motivation professionnelle et sa version humoristique "anti-motivation""
    expect(
      screen.getByText(/Générez instantanément une lettre de motivation/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/version humoristique "anti-motivation"/i)
    ).toBeInTheDocument();
  });

  it('should render AccessGate component', async () => {
    render(await LettersPage(defaultParams));

    const accessGate = screen.getByTestId('access-gate');
    expect(accessGate).toBeInTheDocument();
  });

  it('should render LetterGenerator inside AccessGate', async () => {
    render(await LettersPage(defaultParams));

    const letterGenerator = screen.getByTestId('letter-generator');
    expect(letterGenerator).toBeInTheDocument();

    // Verify it's inside AccessGate (via LettersTabs)
    const accessGate = screen.getByTestId('access-gate');
    expect(accessGate).toContainElement(letterGenerator);
  });

  it('should have gradient background classes', async () => {
    const { container } = render(await LettersPage(defaultParams));

    const mainDiv = container.querySelector('.min-h-screen');
    expect(mainDiv).toBeInTheDocument();
    expect(mainDiv).toHaveClass('bg-gradient-to-br');
  });

  it('should render container with proper classes', async () => {
    const { container } = render(await LettersPage(defaultParams));

    const containerDiv = container.querySelector('.container');
    expect(containerDiv).toBeInTheDocument();
    expect(containerDiv).toHaveClass('mx-auto', 'px-4', 'py-12');
  });

  it('should render heading with proper font weight', async () => {
    render(await LettersPage(defaultParams));

    const heading = screen.getByRole('heading', {
      name: /Générateur de Lettres par IA/i,
    });
    // Le titre h1 de la page lettres a la classe font-bold
    expect(heading).toHaveClass('font-bold');
  });

  it('should have responsive text sizing', async () => {
    render(await LettersPage(defaultParams));

    const heading = screen.getByRole('heading', {
      name: /Générateur de Lettres par IA/i,
    });
    expect(heading).toHaveClass('text-4xl', 'md:text-5xl');
  });

  it('should render centered header section', async () => {
    const { container } = render(await LettersPage(defaultParams));

    const headerDiv = container.querySelector('.text-center.mb-12');
    expect(headerDiv).toBeInTheDocument();
  });

  it('should render description with proper styling', async () => {
    const { container } = render(await LettersPage(defaultParams));

    const description = screen.getByText(/Générez instantanément/i);
    expect(description).toHaveClass('text-lg');
  });
});

describe('LettersPage - Metadata', () => {
  it('should have correct title', async () => {
    // messages.letters.title = "Générateur de Lettres par IA" → "Générateur de Lettres par IA | maicivy"
    const metadata = await generateMetadata({ params: { locale: 'fr' } });
    expect(metadata.title).toBe('Générateur de Lettres par IA | maicivy');
  });

  it('should have correct description', async () => {
    // messages.letters.subtitle = "Générez instantanément une lettre de motivation..."
    const metadata = await generateMetadata({ params: { locale: 'fr' } });
    expect(metadata.description).toContain(
      'Générez instantanément'
    );
    expect(metadata.description).toContain('lettre de motivation');
  });

  it('should have OpenGraph metadata', async () => {
    // openGraph.title = messages.letters.title = "Générateur de Lettres par IA"
    const metadata = await generateMetadata({ params: { locale: 'fr' } });
    expect(metadata.openGraph).toBeDefined();
    expect(metadata.openGraph?.title).toBe('Générateur de Lettres par IA');
    expect(metadata.openGraph?.description).toContain(
      'Générez instantanément'
    );
  });
});
