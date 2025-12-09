import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LetterGenerator } from '../LetterGenerator';
import { server } from '@/__mocks__/server';
import { http, HttpResponse } from 'msw';
import { mockLetterResponse } from '@/lib/testutil/fixtures';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}));

// Mock LetterPreview component
jest.mock('../LetterPreview', () => ({
  LetterPreview: ({ letters, onReset }: any) => (
    <div data-testid="letter-preview">
      <div>{letters.companyName}</div>
      <button onClick={onReset}>Reset</button>
    </div>
  ),
}));

// Setup MSW
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
  localStorage.clear();
});
afterAll(() => server.close());

describe('LetterGenerator', () => {
  it('should render form with company name input', () => {
    render(<LetterGenerator />);

    expect(screen.getByLabelText(/Nom de l'entreprise/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Ex: Google, Microsoft/i)).toBeInTheDocument();
  });

  it('should render submit button', () => {
    render(<LetterGenerator />);

    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });
    expect(submitButton).toBeInTheDocument();
    expect(submitButton).not.toBeDisabled();
  });

  it('should validate company name input - too short', async () => {
    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'A' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Le nom doit contenir au moins 2 caractères/i)
      ).toBeInTheDocument();
    });
  });

  it('should validate company name input - invalid characters', async () => {
    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Company<script>' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Caractères invalides détectés/i)
      ).toBeInTheDocument();
    });
  });

  it('should submit form with valid data and show loading state', async () => {
    server.use(
      http.post('*/api/letters/generate', async () => {
        // Simulate delay
        await new Promise((resolve) => setTimeout(resolve, 100));
        return HttpResponse.json(mockLetterResponse);
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Tech Innovations Inc' } });
    fireEvent.click(submitButton);

    // Should show loading state
    await waitFor(() => {
      expect(screen.getByText(/Génération en cours/i)).toBeInTheDocument();
    });

    // Button should be disabled
    expect(submitButton).toBeDisabled();
  });

  it('should display progress bar during generation', async () => {
    server.use(
      http.post('*/api/letters/generate', async () => {
        await new Promise((resolve) => setTimeout(resolve, 100));
        return HttpResponse.json(mockLetterResponse);
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Google' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/Analyse de l'entreprise/i)).toBeInTheDocument();
    });

    // Progress percentage should be visible
    expect(screen.getByText(/\d+%/)).toBeInTheDocument();
  });

  it('should handle successful generation and show preview', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(mockLetterResponse);
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Tech Innovations Inc' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByTestId('letter-preview')).toBeInTheDocument();
    });

    // Form should be hidden
    expect(screen.queryByLabelText(/Nom de l'entreprise/i)).not.toBeInTheDocument();

    // Preview should show company name
    expect(screen.getByText('Tech Innovations Inc')).toBeInTheDocument();
  });

  it('should handle 403 error (access denied)', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(
          { message: 'Access denied' },
          { status: 403 }
        );
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Google' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Accès refusé. Vous devez effectuer 3 visites/i)
      ).toBeInTheDocument();
    });
  });

  it('should handle 429 error (rate limit)', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(
          { message: 'Rate limit exceeded' },
          { status: 429 }
        );
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Google' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Limite atteinte. Réessayez dans quelques minutes/i)
      ).toBeInTheDocument();
    });
  });

  it('should handle 500 error (server error)', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(
          { message: 'Internal server error' },
          { status: 500 }
        );
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Google' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Erreur serveur. Nos IA prennent une pause café/i)
      ).toBeInTheDocument();
    });
  });

  it('should save generation to localStorage history', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(mockLetterResponse);
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Tech Innovations Inc' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByTestId('letter-preview')).toBeInTheDocument();
    });

    // Check localStorage
    const history = JSON.parse(localStorage.getItem('letters_history') || '[]');
    expect(history).toHaveLength(1);
    expect(history[0].companyName).toBe('Tech Innovations Inc');
  });

  it('should reset form when onReset is called from preview', async () => {
    server.use(
      http.post('*/api/letters/generate', () => {
        return HttpResponse.json(mockLetterResponse);
      })
    );

    render(<LetterGenerator />);

    const input = screen.getByLabelText(/Nom de l'entreprise/i);
    const submitButton = screen.getByRole('button', { name: /Générer les lettres/i });

    fireEvent.change(input, { target: { value: 'Google' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByTestId('letter-preview')).toBeInTheDocument();
    });

    // Click reset button from preview
    const resetButton = screen.getByText('Reset');
    fireEvent.click(resetButton);

    // Form should be visible again
    expect(screen.getByLabelText(/Nom de l'entreprise/i)).toBeInTheDocument();
    expect(screen.queryByTestId('letter-preview')).not.toBeInTheDocument();
  });

  it('should display info message about dual letter generation', () => {
    render(<LetterGenerator />);

    expect(
      screen.getByText(
        /L'IA va générer deux lettres : une motivation professionnelle et une anti-motivation humoristique/i
      )
    ).toBeInTheDocument();
  });
});
