import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminLettersTool from '../page';

// Mock fetch routé par URL : GET /api/v1/ (session), POST generate (→ job_id), GET job (→ completed).
function mockFlow({ jobStatus = 'completed' }: { jobStatus?: string } = {}) {
  global.fetch = jest.fn((url: RequestInfo | URL) => {
    const u = String(url);
    if (u.endsWith('/api/v1/')) return Promise.resolve({ ok: true } as Response);
    if (u.includes('/letters/generate')) {
      return Promise.resolve({ status: 202, ok: true, json: () => Promise.resolve({ job_id: 'job-123' }) } as unknown as Response);
    }
    if (u.includes('/letters/job/')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          status: jobStatus,
          progress: 100,
          letter_motivation_id: 'mot-1',
          letter_anti_motivation_id: 'anti-1',
          error: jobStatus === 'failed' ? 'boom' : undefined,
        }),
      } as unknown as Response);
    }
    return Promise.resolve({ ok: false } as Response);
  }) as jest.Mock;
}

describe('AdminLettersTool', () => {
  afterEach(() => jest.restoreAllMocks());

  it('bouton désactivé sans nom d’entreprise', () => {
    mockFlow();
    render(<AdminLettersTool />);
    expect(screen.getByTestId('lt-generate')).toBeDisabled();
    fireEvent.change(screen.getByTestId('lt-company'), { target: { value: 'Stripe' } });
    expect(screen.getByTestId('lt-generate')).toBeEnabled();
  });

  it('flux complet : session → generate → poll → liens PDF de la paire', async () => {
    mockFlow({ jobStatus: 'completed' });
    render(<AdminLettersTool />);
    fireEvent.change(screen.getByTestId('lt-company'), { target: { value: 'Stripe' } });
    fireEvent.change(screen.getByTestId('lt-offer'), { target: { value: 'Backend Go, paiements' } });
    fireEvent.click(screen.getByTestId('lt-generate'));

    // POST generate appelé avec les bons champs + credentials
    await waitFor(() => {
      const calls = (global.fetch as jest.Mock).mock.calls.map((c) => String(c[0]));
      expect(calls.some((u) => u.includes('/letters/generate'))).toBe(true);
    });
    const genCall = (global.fetch as jest.Mock).mock.calls.find((c) => String(c[0]).includes('/letters/generate'));
    expect(genCall[1].credentials).toBe('include');
    const body = JSON.parse(genCall[1].body);
    expect(body.company_name).toBe('Stripe');
    expect(body.lang).toBe('fr');

    // Le polling aboutit → liens PDF des 2 lettres
    const motiv = await screen.findByTestId('lt-pdf-motivation');
    expect(motiv).toHaveAttribute('href', expect.stringContaining('/letters/mot-1/pdf'));
    expect(screen.getByTestId('lt-pdf-anti')).toHaveAttribute('href', expect.stringContaining('/letters/anti-1/pdf'));
  });

  it('job failed → message d’erreur', async () => {
    mockFlow({ jobStatus: 'failed' });
    render(<AdminLettersTool />);
    fireEvent.change(screen.getByTestId('lt-company'), { target: { value: 'Stripe' } });
    fireEvent.click(screen.getByTestId('lt-generate'));
    expect(await screen.findByTestId('lt-error')).toHaveTextContent(/boom/);
  });
});
