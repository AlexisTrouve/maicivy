import { render, screen, fireEvent, waitFor } from '@testing-library/react';

// Locale 'en' pour ce fichier uniquement (le mock global next-intl force 'fr', langue source —
// aucune UI de traduction ne s'afficherait). requireActual passe par le moduleNameMapper de
// jest.config.js (qui pointe déjà next-intl -> __mocks__/next-intl.tsx), donc on récupère bien le
// mock existant et on ne surcharge QUE useLocale.
jest.mock('next-intl', () => {
  const mock = jest.requireActual('next-intl');
  return { ...mock, useLocale: () => 'en' };
});

import AdminMailboxTool from '../page';

const email1 = {
  id: 'm1',
  from_address: 'notif@malt.fr',
  from_domain: 'malt.fr',
  platform: 'malt',
  subject: 'Mission Go dispo',
  received_at: '2026-07-01T10:00:00Z',
  read: false,
  forwarded_at: null,
  forward_error: '',
};

function mockFetchWithTranslation(opts: {
  cached?: { subject: string; body: string } | null;
  translateResponse?: { subject: string; body: string };
  translateStatus?: number;
}) {
  const cached = opts.cached ?? null;
  const calls: string[] = [];

  global.fetch = jest.fn((url: RequestInfo | URL, fetchOpts?: RequestInit) => {
    const u = String(url);
    const method = fetchOpts?.method || 'GET';
    calls.push(`${method} ${u}`);

    if (u.includes('/translation') && method === 'GET') {
      if (cached) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(cached) } as Response);
      }
      return Promise.resolve({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'not translated yet' }),
      } as Response);
    }
    if (u.includes('/translation') && method === 'POST') {
      const status = opts.translateStatus ?? 200;
      return Promise.resolve({
        ok: status < 300,
        status,
        json: () => Promise.resolve(opts.translateResponse ?? { error: 'fail' }),
      } as Response);
    }
    if (u.includes('/admin/mailbox/') && method === 'GET') {
      return Promise.resolve({
        ok: true,
        json: () =>
          Promise.resolve({
            ...email1,
            message_id: 'm1@malt.fr',
            imap_uid: 1,
            body_text: 'Corps FR original',
          }),
      } as Response);
    }
    if (u.includes('/admin/mailbox') && method === 'GET') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [email1], total: 1, page: 1, per_page: 20, total_pages: 1 }),
      } as Response);
    }
    return Promise.resolve({ ok: false } as Response);
  }) as jest.Mock;

  return calls;
}

describe('AdminMailboxTool — traduction à la demande', () => {
  afterEach(() => jest.restoreAllMocks());

  it('pas de traduction en cache → bouton "Traduire" affiché, ZÉRO appel LLM automatique', async () => {
    const calls = mockFetchWithTranslation({ cached: null });
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));

    await waitFor(() => expect(screen.getByTestId('mailbox-translate-button')).toBeInTheDocument());
    expect(screen.getByTestId('mailbox-body')).toHaveTextContent('Corps FR original');

    const postCalls = calls.filter((c) => c.startsWith('POST') && c.includes('/translation'));
    expect(postCalls).toHaveLength(0);
  });

  it('traduction déjà en cache → servie automatiquement, sans clic', async () => {
    mockFetchWithTranslation({ cached: { subject: 'EN subject', body: 'EN body' } });
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));

    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('EN body'));
    expect(screen.getByText('EN subject')).toBeInTheDocument();
    // Le bouton devient un toggle "voir l'original" (pas "Traduire", déjà traduit)
    expect(screen.getByTestId('mailbox-toggle-translation')).toBeInTheDocument();
    expect(screen.queryByTestId('mailbox-translate-button')).not.toBeInTheDocument();
  });

  it('clic "Traduire" déclenche la traduction et affiche le résultat', async () => {
    mockFetchWithTranslation({
      cached: null,
      translateResponse: { subject: 'EN subject', body: 'EN body' },
    });
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));
    await waitFor(() => expect(screen.getByTestId('mailbox-translate-button')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('mailbox-translate-button'));

    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('EN body'));
    expect(screen.getByText('EN subject')).toBeInTheDocument();
  });

  it('toggle "voir l\'original" / "voir la traduction" bascule le contenu affiché', async () => {
    mockFetchWithTranslation({ cached: { subject: 'EN subject', body: 'EN body' } });
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));
    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('EN body'));

    fireEvent.click(screen.getByTestId('mailbox-toggle-translation'));
    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('Corps FR original'));

    fireEvent.click(screen.getByTestId('mailbox-toggle-translation'));
    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('EN body'));
  });

  it('échec de traduction → message d\'erreur affiché, bouton "Traduire" reste disponible', async () => {
    // 503 (non-4xx) déclenche le retry-with-backoff du client API (~6s cumulés) avant de rejeter —
    // comportement normal du client, pas un bug ; timeout étendu en conséquence.
    mockFetchWithTranslation({ cached: null, translateStatus: 503 });
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));
    await waitFor(() => expect(screen.getByTestId('mailbox-translate-button')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('mailbox-translate-button'));

    await waitFor(() => expect(screen.getByTestId('mailbox-translation-error')).toBeInTheDocument(), {
      timeout: 10000,
    });
    expect(screen.getByTestId('mailbox-translate-button')).toBeInTheDocument();
  }, 15000);
});
