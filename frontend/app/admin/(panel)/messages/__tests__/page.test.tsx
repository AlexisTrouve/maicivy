import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminMessagesTool from '../page';

describe('AdminMessagesTool', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
    Object.assign(navigator, { clipboard: { writeText: jest.fn().mockResolvedValue(undefined) } });
  });
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('bouton désactivé tant que la mission est trop courte', () => {
    render(<AdminMessagesTool />);
    expect(screen.getByTestId('msg-generate')).toBeDisabled();
    fireEvent.change(screen.getByTestId('msg-mission'), { target: { value: 'court' } });
    expect(screen.getByTestId('msg-generate')).toBeDisabled();
    fireEvent.change(screen.getByTestId('msg-mission'), {
      target: { value: 'Mission de développement backend Go pour une plateforme SaaS' },
    });
    expect(screen.getByTestId('msg-generate')).toBeEnabled();
  });

  it('génère via POST /messages/generate (owner cookie) et affiche le message', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: 'Bonjour, votre mission Go m’intéresse…' }),
    });
    render(<AdminMessagesTool />);

    fireEvent.change(screen.getByTestId('msg-mission'), {
      target: { value: 'Développement backend Go, microservices, Kubernetes, Redis' },
    });
    fireEvent.change(screen.getByTestId('msg-platform'), { target: { value: 'linkedin' } });
    fireEvent.change(screen.getByTestId('msg-tjm'), { target: { value: '600' } });
    fireEvent.click(screen.getByTestId('msg-generate'));

    await waitFor(() => expect(global.fetch).toHaveBeenCalled());
    const [url, opts] = (global.fetch as jest.Mock).mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/messages\/generate$/);
    expect(opts.credentials).toBe('include');
    const body = JSON.parse(opts.body);
    expect(body.platform).toBe('linkedin');
    expect(body.tjm).toBe(600);
    expect(body.mission).toContain('Développement backend Go');

    expect(await screen.findByTestId('msg-result')).toHaveTextContent(/mission Go m/);

    // copie
    fireEvent.click(screen.getByTestId('msg-copy'));
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalled());
  });

  it('affiche le message d’erreur backend sur échec', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: 'LLM indisponible' }),
    });
    render(<AdminMessagesTool />);
    fireEvent.change(screen.getByTestId('msg-mission'), {
      target: { value: 'Développement backend Go microservices Kubernetes' },
    });
    fireEvent.click(screen.getByTestId('msg-generate'));
    expect(await screen.findByTestId('msg-error')).toHaveTextContent(/LLM indisponible/);
  });
});
