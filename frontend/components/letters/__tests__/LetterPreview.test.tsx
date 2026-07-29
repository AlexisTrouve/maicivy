import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LetterPreview } from '../LetterPreview';
import { server } from '@/__mocks__/server';
import { rest } from 'msw';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Download: ({ className }: any) => <div data-testid="download-icon" className={className} />,
  RotateCcw: ({ className }: any) => <div data-testid="rotate-icon" className={className} />,
  Copy: ({ className }: any) => <div data-testid="copy-icon" className={className} />,
  Check: ({ className }: any) => <div data-testid="check-icon" className={className} />,
}));

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}));

// Mock lettersApi
jest.mock('@/lib/api', () => ({
  lettersApi: {
    downloadPDF: jest.fn(),
  },
}));

import { lettersApi } from '@/lib/api';

// Setup MSW
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
});
afterAll(() => server.close());

describe('LetterPreview', () => {
  const mockLetters = {
    id: 'letter-123',
    companyName: 'Tech Innovations Inc',
    motivationLetter: 'Dear Hiring Manager,\n\nI am very interested in this position...',
    antiMotivationLetter: 'Dear Sir/Madam,\n\nI reluctantly submit this application...',
    companyInfo: {
      industry: 'Technology',
    },
    createdAt: '2024-01-15T10:00:00Z',
  };

  const mockOnReset = jest.fn();

  beforeEach(() => {
    // Mock clipboard API
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn(() => Promise.resolve()),
      },
    });

    // Mock URL.createObjectURL
    global.URL.createObjectURL = jest.fn(() => 'blob:mock-url');
    global.URL.revokeObjectURL = jest.fn();
  });

  it('should render company name and info', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    expect(screen.getByText(/Lettres pour Tech Innovations Inc/i)).toBeInTheDocument();
    expect(screen.getByText(/Secteur: Technology/i)).toBeInTheDocument();
  });

  it('should display both letters side by side', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    // Motivation letter
    const motivationHeadings = screen.getAllByText(/Lettre de Motivation/i);
    expect(motivationHeadings.length).toBeGreaterThan(0);
    expect(
      screen.getByText(/I am very interested in this position/i)
    ).toBeInTheDocument();

    // Anti-motivation letter
    const antiMotivationHeadings = screen.getAllByText(/Lettre d'Anti-Motivation/i);
    expect(antiMotivationHeadings.length).toBeGreaterThan(0);
    expect(
      screen.getByText(/I reluctantly submit this application/i)
    ).toBeInTheDocument();
  });

  it('should render action buttons', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    expect(screen.getByText(/PDF Dual/i)).toBeInTheDocument();
    expect(screen.getByText(/Nouvelle génération/i)).toBeInTheDocument();
  });

  it('should call onReset when reset button is clicked', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const resetButton = screen.getByText(/Nouvelle génération/i);
    fireEvent.click(resetButton);

    expect(mockOnReset).toHaveBeenCalledTimes(1);
  });

  it('should copy motivation letter to clipboard', async () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    // Find copy buttons (there are 2, one for each letter)
    const copyButtons = screen.getAllByTitle(/Copier le texte/i);
    const motivationCopyButton = copyButtons[0];

    fireEvent.click(motivationCopyButton);

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        mockLetters.motivationLetter
      );
    });

    // Check icon should appear briefly
    const checkIcon = screen.getAllByTitle(/Copier le texte/i)[0];
    expect(checkIcon).toBeInTheDocument();
  });

  it('should copy anti-motivation letter to clipboard', async () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const copyButtons = screen.getAllByTitle(/Copier le texte/i);
    const antiCopyButton = copyButtons[1];

    fireEvent.click(antiCopyButton);

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        mockLetters.antiMotivationLetter
      );
    });
  });

  it('should download PDF dual when button is clicked', async () => {
    const mockBlob = new Blob(['pdf content'], { type: 'application/pdf' });
    (lettersApi.downloadPDF as jest.Mock).mockResolvedValue(mockBlob);

    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const pdfDualButton = screen.getByText(/PDF Dual/i);
    fireEvent.click(pdfDualButton);

    await waitFor(() => {
      expect(lettersApi.downloadPDF).toHaveBeenCalledWith('letter-123', 'both');
    });
  });

  it('should download motivation PDF individually', async () => {
    const mockBlob = new Blob(['pdf content'], { type: 'application/pdf' });
    (lettersApi.downloadPDF as jest.Mock).mockResolvedValue(mockBlob);

    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const downloadButtons = screen.getAllByTitle(/Télécharger PDF/i);
    const motivationDownloadButton = downloadButtons[0];

    fireEvent.click(motivationDownloadButton);

    await waitFor(() => {
      expect(lettersApi.downloadPDF).toHaveBeenCalledWith('letter-123', 'motivation');
    });
  });

  it('should download anti-motivation PDF individually', async () => {
    const mockBlob = new Blob(['pdf content'], { type: 'application/pdf' });
    (lettersApi.downloadPDF as jest.Mock).mockResolvedValue(mockBlob);

    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const downloadButtons = screen.getAllByTitle(/Télécharger PDF/i);
    const antiDownloadButton = downloadButtons[1];

    fireEvent.click(antiDownloadButton);

    await waitFor(() => {
      expect(lettersApi.downloadPDF).toHaveBeenCalledWith('letter-123', 'anti');
    });
  });

  it('should show loading state during PDF download', async () => {
    (lettersApi.downloadPDF as jest.Mock).mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve(new Blob()), 100))
    );

    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    const pdfDualButton = screen.getByText(/PDF Dual/i);
    fireEvent.click(pdfDualButton);

    // Should show spinner
    await waitFor(() => {
      const spinner = document.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });
  });

  it('should display warning note about anti-motivation letter', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    expect(
      screen.getByText(
        /La lettre d'anti-motivation est générée à titre humoristique/i
      )
    ).toBeInTheDocument();

    expect(
      screen.getByText(/Elle ne doit PAS être envoyée à l'entreprise/i)
    ).toBeInTheDocument();
  });

  it('should render correct icons for each letter type', () => {
    render(<LetterPreview letters={mockLetters} onReset={mockOnReset} />);

    // Motivation letter should have checkmark
    expect(screen.getByText('✅')).toBeInTheDocument();

    // Anti-motivation letter should have X
    expect(screen.getByText('❌')).toBeInTheDocument();
  });

  it('should apply correct gradient colors to headers', () => {
    const { container } = render(
      <LetterPreview letters={mockLetters} onReset={mockOnReset} />
    );

    // Motivation: green gradient
    const motivationHeader = container.querySelector('.from-green-500');
    expect(motivationHeader).toBeInTheDocument();

    // Anti-motivation: orange/red gradient
    const antiHeader = container.querySelector('.from-orange-500');
    expect(antiHeader).toBeInTheDocument();
  });
});
