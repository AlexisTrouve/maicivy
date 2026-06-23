import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminLoginPage from '../page';

// useRouter (next/navigation) est fourni par le mock global (jest.config). On mocke fetch par test.
describe('AdminLoginPage', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('poste le mot de passe au backend (POST /admin/login, credentials include)', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({ ok: true });
    render(<AdminLoginPage />);

    fireEvent.change(screen.getByTestId('admin-password'), { target: { value: 's3cret' } });
    fireEvent.click(screen.getByTestId('admin-submit'));

    await waitFor(() => expect(global.fetch).toHaveBeenCalled());
    const [url, opts] = (global.fetch as jest.Mock).mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/admin\/login$/);
    expect(opts.method).toBe('POST');
    expect(opts.credentials).toBe('include');
    expect(JSON.parse(opts.body)).toEqual({ password: 's3cret' });
  });

  it('affiche une erreur sur mot de passe invalide (réponse non-ok)', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({ ok: false });
    render(<AdminLoginPage />);

    fireEvent.change(screen.getByTestId('admin-password'), { target: { value: 'wrong' } });
    fireEvent.click(screen.getByTestId('admin-submit'));

    expect(await screen.findByTestId('admin-error')).toHaveTextContent(/invalide/i);
  });
});
