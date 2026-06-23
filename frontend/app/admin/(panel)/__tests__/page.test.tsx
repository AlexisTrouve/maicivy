import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminCVTool from '../page';

describe('AdminCVTool', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
    // jsdom n'a pas createObjectURL/revokeObjectURL → on les stub.
    global.URL.createObjectURL = jest.fn(() => 'blob:mock-url');
    global.URL.revokeObjectURL = jest.fn();
  });
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('le bouton est désactivé tant que l’offre est trop courte', () => {
    render(<AdminCVTool />);
    expect(screen.getByTestId('cv-generate')).toBeDisabled();
    fireEvent.change(screen.getByTestId('cv-offer'), { target: { value: 'trop court' } });
    expect(screen.getByTestId('cv-generate')).toBeDisabled();
    fireEvent.change(screen.getByTestId('cv-offer'), {
      target: { value: 'Développeur Go backend microservices Redis PostgreSQL' },
    });
    expect(screen.getByTestId('cv-generate')).toBeEnabled();
  });

  it('génère via POST /cv/generate (format pdf, credentials include) et affiche la preview', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['%PDF-'])),
    });
    render(<AdminCVTool />);

    fireEvent.change(screen.getByTestId('cv-offer'), {
      target: { value: 'Lead backend Go, microservices event-driven, Kafka, observabilité' },
    });
    fireEvent.click(screen.getByTestId('cv-generate'));

    await waitFor(() => expect(global.fetch).toHaveBeenCalled());
    const [url, opts] = (global.fetch as jest.Mock).mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/cv\/generate$/);
    expect(opts.method).toBe('POST');
    expect(opts.credentials).toBe('include');
    const body = JSON.parse(opts.body);
    expect(body.format).toBe('pdf');
    expect(body.lang).toBe('fr');
    expect(body.offer).toContain('Lead backend Go');

    // preview + bouton de téléchargement apparaissent
    await waitFor(() => expect(screen.getByTestId('cv-download')).toBeInTheDocument());
    expect(screen.getByTestId('cv-preview')).toHaveAttribute('src', 'blob:mock-url');
  });

  it('affiche le message d’erreur backend sur échec', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: 'LLM generation failed: proxy' }),
    });
    render(<AdminCVTool />);

    fireEvent.change(screen.getByTestId('cv-offer'), {
      target: { value: 'Développeur backend Go microservices Kubernetes Redis' },
    });
    fireEvent.click(screen.getByTestId('cv-generate'));

    expect(await screen.findByTestId('cv-error')).toHaveTextContent(/LLM generation failed/);
  });
});
