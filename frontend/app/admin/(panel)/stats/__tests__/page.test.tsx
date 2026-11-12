import { render, screen, waitFor } from '@testing-library/react';
import AdminStats from '../page';

const DATA = {
  ai: {
    generations_today: 42,
    letters_this_month: [{ model: 'claude-opus-4-6', count: 5, tokens: 12000, cost_eur: 0.42 }],
    letters_tokens: 12000,
    letters_cost_eur: 0.42,
  },
  security: {
    flagged_ips: [{ ip: '2607:5300::1', score: 8.3, paths: ['/.env', '/wp-admin'] }],
    flagged_count: 1,
  },
  analytics: {
    by_profile: [{ profile: 'recruiter', count: 12 }],
    top_referrers: [{ referrer: 'https://linkedin.com', count: 7 }],
  },
};

describe('AdminStats', () => {
  afterEach(() => jest.restoreAllMocks());

  it('charge /admin/stats et rend les 3 sections', async () => {
    global.fetch = jest.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(DATA) });
    render(<AdminStats />);

    await waitFor(() => expect(screen.getByTestId('admin-stats')).toBeInTheDocument());
    // appel avec credentials
    const [url, opts] = (global.fetch as jest.Mock).mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/admin\/stats$/);
    expect(opts.credentials).toBe('include');

    // section IA
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByTestId('stats-ai-table')).toHaveTextContent('claude-opus-4-6');
    // section sécurité
    expect(screen.getByTestId('stats-sus-list')).toHaveTextContent('2607:5300::1');
    expect(screen.getByTestId('stats-sus-list')).toHaveTextContent('/.env');
    // section analytics
    expect(screen.getByTestId('stats-profiles')).toHaveTextContent('recruiter');
    expect(screen.getByTestId('stats-referrers')).toHaveTextContent('linkedin');
  });

  it('affiche une erreur si le chargement échoue', async () => {
    global.fetch = jest.fn().mockResolvedValue({ ok: false });
    render(<AdminStats />);
    expect(await screen.findByTestId('stats-error')).toBeInTheDocument();
  });
});
