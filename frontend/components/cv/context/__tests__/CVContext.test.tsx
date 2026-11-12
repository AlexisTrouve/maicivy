import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { CVContextProvider } from '../CVContextProvider';
import { CVContextBanner } from '../CVContextBanner';
import { CVContextContent } from '../CVContextContent';
import { AdaptToOfferForm } from '../AdaptToOfferForm';

// QUOI : le CV adapté sur /cv, de bout en bout côté composants (le réseau est simulé).
//   - sans contexte : aucun bandeau, aucune question, le standard ;
//   - en préparation : bandeau + standard, puis bascule automatique sur l'adapté quand il arrive ;
//   - prêt : bascule adapté ↔ standard, téléchargement du PDF adapté (cookie inclus) ;
//   - indisponible (quota) : bandeau sobre, standard ;
//   - « l'adapter à une offre » réutilise POST /cv/generate (format pdf), quota nommé.

jest.mock('@/components/cv/ExperienceTimeline', () => ({
  __esModule: true,
  default: ({ experiences }: { experiences: { company: string }[] }) => (
    <div data-testid="timeline">{experiences.map((e) => e.company).join(',')}</div>
  ),
}));
jest.mock('@/components/cv/SkillsCloud', () => ({ __esModule: true, default: () => <div /> }));
jest.mock('@/components/cv/ProjectsGrid', () => ({ __esModule: true, default: () => <div /> }));

const adaptedCV = { theme: 'dynamic', experiences: [{ company: 'AdaptéCorp' }], skills: [], projects: [] };
const NEED = 'Lead Go pour une plateforme d’agents IA';

const json = (body: unknown, status = 200) =>
  Promise.resolve(new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }));

function renderPage() {
  return render(
    <CVContextProvider>
      <CVContextBanner />
      <AdaptToOfferForm />
      <CVContextContent>
        <div data-testid="standard-body">standard</div>
      </CVContextContent>
    </CVContextProvider>
  );
}

describe('CV adapté au besoin du visiteur', () => {
  let fetchMock: jest.Mock;
  beforeEach(() => {
    fetchMock = jest.fn();
    global.fetch = fetchMock;
    global.URL.createObjectURL = jest.fn(() => 'blob:x');
    global.URL.revokeObjectURL = jest.fn();
  });
  afterEach(() => jest.useRealTimers());

  it('sans contexte : le CV standard, aucun bandeau, aucune question', async () => {
    fetchMock.mockImplementation(() => json({ status: 'none' }));
    renderPage();
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(fetchMock.mock.calls[0][0]).toContain('/api/v1/cv/context?lang=fr');
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
    expect(screen.queryByTestId('cv-context-banner')).toBeNull();
    expect(screen.getByTestId('cv-standard')).toBeVisible();
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(screen.getByTestId('cv-offer-toggle')).toHaveTextContent("L'adapter à une offre");
  });

  it('en préparation puis prêt : le standard reste, puis l’adapté le remplace', async () => {
    // Seuls les timers sont simulés : Response.json() a besoin des vraies microtâches (les simuler
    // laisse findBy* tourner à vide jusqu'à épuiser la mémoire du worker).
    jest.useFakeTimers({ doNotFake: ['queueMicrotask', 'nextTick', 'setImmediate'] });
    fetchMock
      .mockImplementationOnce(() => json({ status: 'pending', need: NEED, source: 'chat', lang: 'fr' }))
      .mockImplementation(() => json({ status: 'ready', need: NEED, source: 'chat', lang: 'fr', cv: adaptedCV }));
    renderPage();

    expect(await screen.findByTestId('cv-context-pending')).toBeInTheDocument();
    expect(screen.getByText("Adapté à ce que vous m'avez dit")).toBeInTheDocument();
    expect(screen.getByText("D'après notre conversation")).toBeInTheDocument();
    expect(screen.getByTestId('cv-context-need')).toHaveTextContent(NEED);
    expect(screen.getByTestId('cv-standard')).toBeVisible();

    await act(async () => {
      jest.advanceTimersByTime(4000);
    });
    expect(await screen.findByTestId('cv-adapted')).toBeInTheDocument();
    expect(screen.getByTestId('timeline')).toHaveTextContent('AdaptéCorp');
    expect(screen.getByTestId('cv-standard')).not.toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('prêt : bascule standard ↔ adapté et télécharge le PDF adapté', async () => {
    fetchMock.mockImplementation((url: string) =>
      url.includes('/context/pdf')
        ? Promise.resolve(
            new Response('%PDF', { status: 200, headers: { 'Content-Disposition': 'attachment; filename="cv_adapte_fr.pdf"' } })
          )
        : json({ status: 'ready', need: NEED, source: 'dossier', lang: 'fr', cv: adaptedCV })
    );
    renderPage();

    const adaptedBtn = await screen.findByTestId('cv-view-adapted');
    expect(screen.getByText("D'après votre dossier")).toBeInTheDocument();
    expect(adaptedBtn).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(screen.getByTestId('cv-view-standard'));
    expect(screen.queryByTestId('cv-adapted')).toBeNull();
    expect(screen.getByTestId('cv-standard')).toBeVisible();
    expect(screen.getByTestId('cv-view-standard')).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(adaptedBtn);
    expect(screen.getByTestId('cv-adapted')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('cv-context-download'));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/api/v1/cv/context/pdf?lang=fr'), { credentials: 'include' })
    );
    await waitFor(() => expect(global.URL.createObjectURL).toHaveBeenCalled());
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('quota atteint : bandeau sobre, le standard reste', async () => {
    fetchMock.mockImplementation(() => json({ status: 'limited', need: NEED, source: 'chat', lang: 'fr' }));
    renderPage();
    expect(await screen.findByTestId('cv-context-unavailable')).toBeInTheDocument();
    expect(screen.queryByTestId('cv-view-adapted')).toBeNull();
    expect(screen.getByTestId('cv-standard')).toBeVisible();
  });

  it('adapter à une offre : POST /cv/generate en PDF, quota nommé', async () => {
    fetchMock.mockImplementation((url: string) =>
      url.includes('/cv/generate') ? json({ code: 'DAILY_LIMIT_REACHED' }, 429) : json({ status: 'none' })
    );
    renderPage();
    fireEvent.click(screen.getByTestId('cv-offer-toggle'));
    expect(screen.getByTestId('cv-offer-submit')).toBeDisabled();
    fireEvent.change(screen.getByTestId('cv-offer-input'), { target: { value: 'Développeur Go senior, Paris' } });
    fireEvent.click(screen.getByTestId('cv-offer-submit'));

    expect(await screen.findByRole('alert')).toHaveTextContent("Limite de générations atteinte pour aujourd'hui");
    const call = fetchMock.mock.calls.find(([u]) => String(u).includes('/cv/generate'));
    expect(call[1]).toMatchObject({ method: 'POST', credentials: 'include' });
    expect(JSON.parse(call[1].body)).toEqual({ offer: 'Développeur Go senior, Paris', lang: 'fr', format: 'pdf' });
  });
});
