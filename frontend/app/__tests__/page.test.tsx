import { render, screen } from '@testing-library/react';

import HomePage, { dynamic, generateMetadata } from '../[locale]/page';
import { getMontagne } from '@/lib/montagne-api';

/**
 * Accueil — depuis la refonte de 2026-11, l'accueil EST la montagne (ex-/[locale]/montagne).
 *
 * Verrouille ce que l'E2E ne lit pas commodément : le JSON-LD Person (positionnement « Agent
 * Orchestrator — Technomancer », profils officiels), le rendu à la demande, les métadonnées de la
 * montagne, et les deux issues du rendu serveur (montagne servie / backend absent). La mise en scène
 * elle-même est couverte en vrai navigateur par tests/e2e/montagne.spec.ts.
 */

jest.mock('@/lib/montagne-api', () => ({ getMontagne: jest.fn() }));

// Les composants de la montagne sont des Server Components async (ou des îlots client lourds) :
// bouchons qui disent seulement qu'ils sont montés.
jest.mock('@/components/montagne/MontagneVoice', () => ({ MontagneVoice: () => <section data-testid="montagne-voice" /> }));
jest.mock('@/components/montagne/MontagneExplorer', () => ({
  MontagneExplorer: ({ children }: { children: React.ReactNode }) => <div data-testid="montagne-explorer">{children}</div>,
}));
jest.mock('@/components/montagne/MontagnePeak', () => ({ MontagnePeak: () => <div data-testid="montagne-peak" /> }));
jest.mock('@/components/montagne/MontagneList', () => ({ MontagneList: () => <div data-testid="montagne-list" /> }));

const params = Promise.resolve({ locale: 'fr' });
const mockedGetMontagne = getMontagne as jest.Mock;

function personSchema(container: HTMLElement) {
  const script = container.querySelector('script[type="application/ld+json"]');
  expect(script).not.toBeNull();
  return JSON.parse(script!.innerHTML);
}

describe('Accueil = la montagne', () => {
  it('rendu à la demande : une recuration maiProFiles apparaît sans redéploiement', () => {
    expect(dynamic).toBe('force-dynamic');
  });

  it('métadonnées : celles de la montagne', async () => {
    const meta = await generateMetadata({ params });
    expect(meta.title).toBe('La montagne — tous les projets reliés');
  });

  it('montagne servie : voix, sommet et liste — et le JSON-LD Person au nouveau positionnement', async () => {
    mockedGetMontagne.mockResolvedValue({ nodes: [], links: [], summit: [], strata: [] });
    const { container } = render(await HomePage({ params }));
    expect(screen.getByTestId('montagne-voice')).toBeInTheDocument();
    expect(screen.getByTestId('montagne-explorer')).toContainElement(screen.getByTestId('montagne-peak'));
    expect(screen.getByTestId('montagne-list')).toBeInTheDocument();
    expect(screen.queryByTestId('montagne-unavailable')).toBeNull();

    const person = personSchema(container);
    expect(person['@type']).toBe('Person');
    expect(person.name).toBe('Alexis Trouvé');
    expect(person.jobTitle).toBe('Agent Orchestrator — Technomancer');
    expect(person.sameAs).toEqual(expect.arrayContaining([expect.stringContaining('linkedin.com'), expect.stringContaining('github.com')]));
  });

  it('backend absent : la voix reste, la montagne dit pourquoi elle manque (jamais une page d’erreur)', async () => {
    mockedGetMontagne.mockRejectedValue(new Error('backend down'));
    const { container } = render(await HomePage({ params }));
    expect(screen.getByTestId('montagne-voice')).toBeInTheDocument();
    expect(screen.getByTestId('montagne-unavailable')).toBeInTheDocument();
    expect(screen.queryByTestId('montagne-explorer')).toBeNull();
    // Le JSON-LD ne dépend pas du backend.
    expect(personSchema(container).jobTitle).toBe('Agent Orchestrator — Technomancer');
  });
});
