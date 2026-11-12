import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { useSearchParams } from 'next/navigation';
import { useRouter as useI18nRouter } from '@/i18n/navigation';
import CVThemeSelector from '../CVThemeSelector';
import { server } from '@/__mocks__/server';
import { rest } from 'msw';

// Mock Next.js navigation hooks
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
  useSearchParams: jest.fn(),
}));

// Mock i18n/navigation (uses next-intl/navigation which is ESM — not transformable)
jest.mock('@/i18n/navigation', () => ({
  useRouter: jest.fn(),
  usePathname: jest.fn(() => '/'),
  Link: ({ children }: { children: React.ReactNode }) => children,
  redirect: jest.fn(),
}));

const mockPush = jest.fn();
const mockSearchParams = new URLSearchParams();

// Forme réelle de GET /cv/themes : { count, themes: [...] }, noms et descriptions de l'API en
// anglais/français mélangés — c'est précisément ce que l'interface ne doit PAS afficher tel quel.
const apiThemes = {
  count: 3,
  themes: [
    { id: 'backend', name: 'Backend Developer', description: 'Focus sur développement backend', icon: '🖥' },
    { id: 'ai', name: 'AI Agent Engineer — Technomancer', description: 'Systèmes multi-agents IA', icon: '🤖' },
    { id: 'quantum', name: 'Quantum Developer', description: 'Profil inconnu du front', icon: '⚛' },
  ],
};

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
});
afterAll(() => server.close());

describe('CVThemeSelector', () => {
  beforeEach(() => {
    (useSearchParams as jest.Mock).mockReturnValue(mockSearchParams);
    (useI18nRouter as jest.Mock).mockReturnValue({ push: mockPush });
    server.use(rest.get('*/api/v1/cv/themes', (_req, res, ctx) => res(ctx.json(apiThemes))));
  });

  // Régression du libellé VIDE : le déclencheur doit montrer le nom du profil courant, et ce dès
  // le premier rendu — sans attendre la liste des profils (il ne dépend plus d'elle).
  it('affiche le nom traduit du profil courant dans le déclencheur, avant même la liste', () => {
    render(<CVThemeSelector currentTheme="ai" />);
    expect(screen.getByRole('combobox')).toHaveTextContent('Agents IA — Technomancer');
  });

  it('désactive le déclencheur tant que la liste des profils charge', async () => {
    render(<CVThemeSelector currentTheme="ai" />);
    expect(screen.getByRole('combobox')).toHaveAttribute('aria-busy', 'true');
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveAttribute('aria-busy', 'false'));
  });

  it('liste les profils traduits, le profil agents IA en premier', async () => {
    render(<CVThemeSelector currentTheme="backend" />);
    const options = await screen.findAllByRole('option');
    expect(options.map((o) => o.getAttribute('data-value'))).toEqual(['ai', 'backend', 'quantum']);
    // Nom et description viennent des messages, pas de l'API (« Backend Developer » / « Focus sur… »).
    expect(options[1]).toHaveTextContent('Backend');
    expect(options[1]).toHaveTextContent('APIs, bases de données, services.');
    expect(options[1]).not.toHaveTextContent('Focus sur');
  });

  it("garde le nom de l'API pour un profil que l'interface ne connaît pas", async () => {
    render(<CVThemeSelector currentTheme="quantum" />);
    const options = await screen.findAllByRole('option');
    expect(options[2]).toHaveTextContent('Quantum Developer');
    expect(screen.getByRole('combobox')).toHaveTextContent('Quantum Developer');
  });

  it('pose l\u2019id demandé sur le déclencheur (nommé par le <label> de la page)', () => {
    render(<CVThemeSelector currentTheme="ai" triggerId="cv-profile" />);
    expect(screen.getByRole('combobox')).toHaveAttribute('id', 'cv-profile');
  });

  it("n'échoue pas si l'API des profils répond en erreur : le libellé reste lisible", async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    server.use(rest.get('*/api/v1/cv/themes', (_req, res, ctx) => res(ctx.status(500))));

    render(<CVThemeSelector currentTheme="ai" />);
    await waitFor(() => expect(screen.getByRole('combobox')).toHaveAttribute('aria-busy', 'false'));
    expect(screen.getByRole('combobox')).toHaveTextContent('Agents IA — Technomancer');
    expect(mockPush).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });
});
