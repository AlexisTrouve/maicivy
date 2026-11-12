import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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
const email2 = {
  id: 'm2',
  from_address: 'notif@malt.fr',
  from_domain: 'malt.fr',
  platform: 'malt',
  subject: 'Autre mission',
  received_at: '2026-07-02T10:00:00Z',
  read: true,
  forwarded_at: '2026-07-02T10:05:00Z',
  forward_error: '',
};
const blockedEmail = {
  id: 'm3',
  from_address: 'notif@malt.fr',
  from_domain: 'malt.fr',
  platform: 'malt',
  subject: 'Mission hors profil',
  received_at: '2026-07-03T10:00:00Z',
  read: false,
  forwarded_at: null,
  forward_error: '',
  is_opportunity: true,
  relevance_score: 25,
  relevance_reason: 'Hors profil',
  relevance_link: 'https://malt.fr/mission/456',
  forward_blocked: true,
};

// Mock fetch routé par URL/méthode : GET liste, GET détail (marque lu côté serveur), POST read, POST forward.
function mockFetch(emails: any[] = [email1, email2]) {
  global.fetch = jest.fn((url: RequestInfo | URL, opts?: RequestInit) => {
    const u = String(url);
    const method = opts?.method || 'GET';

    if (u.includes('/admin/mailbox/') && u.includes('/read') && method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ ok: true }) } as Response);
    }
    if (u.includes('/admin/mailbox/') && u.includes('/forward') && method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ ok: true }) } as Response);
    }
    if (u.includes('/admin/mailbox/') && method === 'GET') {
      const id = u.split('/admin/mailbox/')[1];
      const base = emails.find((e) => e.id === id) || email1;
      const detail = {
        ...base,
        message_id: `${id}@malt.fr`,
        imap_uid: 1,
        body_text: `Corps du mail ${id}`,
        relevance_cot: base.is_opportunity ? `Checked get_experience for ${id}: found relevant background.` : undefined,
      };
      return Promise.resolve({ ok: true, json: () => Promise.resolve(detail) } as Response);
    }
    if (u.includes('/admin/mailbox') && method === 'GET') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails, total: emails.length, page: 1, per_page: 20, total_pages: 1 }),
      } as Response);
    }
    return Promise.resolve({ ok: false } as Response);
  }) as jest.Mock;
}

describe('AdminMailboxTool', () => {
  afterEach(() => jest.restoreAllMocks());

  it('affiche la liste des mails captés', async () => {
    mockFetch();
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    expect(screen.getByTestId('mailbox-item-m2')).toBeInTheDocument();
    expect(screen.getByTestId('mailbox-list')).toHaveTextContent('Mission Go dispo');
  });

  it('liste vide → mailbox-empty', async () => {
    mockFetch([]);
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-empty')).toBeInTheDocument());
  });

  it('sélectionner un mail charge le détail (body_text) dans mailbox-detail', async () => {
    mockFetch();
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('mailbox-item-m1'));

    expect(screen.getByTestId('mailbox-detail')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toHaveTextContent('Corps du mail m1'));
  });

  // Les mails Malt/HubSpot embarquent des URLs de tracking énormes en clair dans le texte brut —
  // illisibles telles quelles. Le corps doit les remplacer par un lien court cliquable (domaine),
  // pas afficher la chaîne de tracking complète.
  it('les URLs de tracking dans le corps deviennent des liens courts cliquables', async () => {
    const longUrl = 'https://d2Z2Gr04.eu1.hubspotlinks.com/Ctc/W3+113/' + 'W'.repeat(200);
    global.fetch = jest.fn((url: RequestInfo | URL, opts?: RequestInit) => {
      const u = String(url);
      const method = opts?.method || 'GET';
      if (u.includes('/admin/mailbox/') && method === 'GET') {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              ...email1,
              message_id: 'm1@malt.fr',
              imap_uid: 1,
              body_text: `Malt (${longUrl}) vous informe.`,
            }),
        } as Response);
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ emails: [email1], total: 1, page: 1, per_page: 20, total_pages: 1 }),
      } as Response);
    }) as jest.Mock;

    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));

    await waitFor(() => expect(screen.getByTestId('mailbox-body-link')).toBeInTheDocument());
    const link = screen.getByTestId('mailbox-body-link');
    expect(link).toHaveAttribute('href', longUrl);
    // Le texte visible est le domaine, jamais la chaîne de tracking complète
    expect(link).toHaveTextContent('d2z2gr04.eu1.hubspotlinks.com');
    expect(link.textContent).not.toContain('W'.repeat(50));
    expect(screen.getByTestId('mailbox-body')).toHaveTextContent('Malt (');
    expect(screen.getByTestId('mailbox-body')).toHaveTextContent(') vous informe.');
  });

  it('bouton "retenter le transfert" appelle POST /forward puis rafraîchit le détail', async () => {
    mockFetch();
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));
    await waitFor(() => expect(screen.getByTestId('mailbox-body')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('mailbox-retry-forward'));

    await waitFor(() => {
      const calls = (global.fetch as jest.Mock).mock.calls.map((c) => `${c[1]?.method || 'GET'} ${c[0]}`);
      expect(calls.some((c) => c.startsWith('POST') && c.includes('/admin/mailbox/m1/forward'))).toBe(true);
    });
  });

  it('bouton marquer lu/non lu appelle POST /read', async () => {
    mockFetch();
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('mailbox-item-m1'));
    await waitFor(() => expect(screen.getByTestId('mailbox-toggle-read')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('mailbox-toggle-read'));

    await waitFor(() => {
      const calls = (global.fetch as jest.Mock).mock.calls.map((c) => `${c[1]?.method || 'GET'} ${c[0]}`);
      expect(calls.some((c) => c.startsWith('POST') && c.includes('/admin/mailbox/m1/read'))).toBe(true);
    });
  });

  it('mail bloqué par le filtre de pertinence → badge dans la liste + verdict et bouton "Forcer l\'envoi" dans le détail', async () => {
    mockFetch([blockedEmail]);
    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m3')).toBeInTheDocument());
    expect(screen.getByTestId('mailbox-blocked-m3')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('mailbox-item-m3'));

    await waitFor(() => expect(screen.getByTestId('mailbox-relevance')).toBeInTheDocument());
    expect(screen.getByTestId('mailbox-relevance')).toHaveTextContent('25/100');
    expect(screen.getByTestId('mailbox-relevance')).toHaveTextContent('Hors profil');
    expect(screen.getByTestId('mailbox-retry-forward')).toHaveTextContent("Forcer l'envoi");

    // Lien mission cliquable, ouvre dans un nouvel onglet
    const link = screen.getByTestId('mailbox-relevance-link');
    expect(link).toHaveAttribute('href', 'https://malt.fr/mission/456');
    expect(link).toHaveAttribute('target', '_blank');

    // Raisonnement (CoT) masqué par défaut, révélé au clic
    expect(screen.queryByTestId('mailbox-cot')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('mailbox-toggle-cot'));
    expect(screen.getByTestId('mailbox-cot')).toHaveTextContent('Checked get_experience for m3');

    fireEvent.click(screen.getByTestId('mailbox-retry-forward'));
    await waitFor(() => {
      const calls = (global.fetch as jest.Mock).mock.calls.map((c) => `${c[1]?.method || 'GET'} ${c[0]}`);
      expect(calls.some((c) => c.startsWith('POST') && c.includes('/admin/mailbox/m3/forward'))).toBe(true);
    });
  });

  it('pagination : le bouton suivant charge la page 2', async () => {
    global.fetch = jest.fn((url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes('/admin/mailbox/')) {
        return Promise.resolve({ ok: false } as Response);
      }
      if (u.includes('/admin/mailbox')) {
        const page = Number(new URL(u).searchParams.get('page') || '1');
        const emails = page === 1 ? [email1] : [email2];
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ emails, total: 2, page, per_page: 20, total_pages: 2 }),
        } as Response);
      }
      return Promise.resolve({ ok: false } as Response);
    }) as jest.Mock;

    render(<AdminMailboxTool />);
    await waitFor(() => expect(screen.getByTestId('mailbox-item-m1')).toBeInTheDocument());
    expect(screen.getByTestId('mailbox-next-page')).toBeEnabled();

    fireEvent.click(screen.getByTestId('mailbox-next-page'));

    await waitFor(() => expect(screen.getByTestId('mailbox-item-m2')).toBeInTheDocument());
  });
});
