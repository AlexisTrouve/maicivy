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

  // Mode prospecteur PAR DÉFAUT (les prospecteurs ne voyaient pas le lien de bascule), focus sur le
  // premier champ du mode actif.
  it('s’ouvre en mode prospecteur, focus sur le nom/email ; la bascule owner focus le mot de passe', async () => {
    render(<AdminLoginPage />);
    expect(screen.getByTestId('prospect-login')).toHaveFocus();
    fireEvent.click(screen.getByTestId('login-mode-switch'));
    expect(screen.queryByTestId('prospect-login')).toBeNull();
    await waitFor(() => expect(screen.getByTestId('admin-password')).toHaveFocus());
  });

  it('mode prospecteur : POST /admin/prospect/login avec login + mot de passe', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({ ok: true });
    render(<AdminLoginPage />);
    fireEvent.change(screen.getByTestId('prospect-login'), { target: { value: 'operatrice' } });
    fireEvent.change(screen.getByTestId('admin-password'), { target: { value: 'pw' } });
    fireEvent.click(screen.getByTestId('admin-submit'));
    await waitFor(() => expect(global.fetch).toHaveBeenCalled());
    const [url, opts] = (global.fetch as jest.Mock).mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/admin\/prospect\/login$/);
    expect(JSON.parse(opts.body)).toEqual({ login: 'operatrice', password: 'pw' });
  });

  it('poste le mot de passe au backend (POST /admin/login, credentials include)', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({ ok: true });
    render(<AdminLoginPage />);
    fireEvent.click(screen.getByTestId('login-mode-switch'));

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
    fireEvent.click(screen.getByTestId('login-mode-switch'));

    fireEvent.change(screen.getByTestId('admin-password'), { target: { value: 'wrong' } });
    fireEvent.click(screen.getByTestId('admin-submit'));

    expect(await screen.findByTestId('admin-error')).toHaveTextContent(/invalide/i);
  });
});
