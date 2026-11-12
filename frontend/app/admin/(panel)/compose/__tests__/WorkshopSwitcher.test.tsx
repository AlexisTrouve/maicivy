import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import WorkshopSwitcher from '../WorkshopSwitcher';

// QUOI : le sélecteur d'atelier de /admin/compose — l'historique de travail de l'opératrice.
//
// POURQUOI ces tests : « nouvel atelier » enterrait le précédent, et l'opératrice traite plusieurs
// annonces par jour puis y revient quand un client répond. Le travail existait en base ; il ne lui
// était simplement pas accessible.
//
// CE QUI EST VÉRIFIÉ ICI est le CÂBLAGE : la liste s'ouvre, la bascule remonte au parent, et les
// actions de métadonnées tapent le bon endpoint avec le bon corps. Le rendu, lui, se regarde en
// capture — jest ne prouve pas l'apparence (leçon du handoff f65a7844).

jest.mock('next-intl', () => ({
  useTranslations: () => (k: string, v?: Record<string, unknown>) =>
    v ? `${k}:${JSON.stringify(v)}` : k,
  useLocale: () => 'en',
}));

const ACTIFS = [
  {
    id: 'w1',
    title: 'Galadrim — AI agents',
    rate: 300,
    has_draft: true,
    checks_pass: 5,
    checks_total: 6,
    updated_at: '2026-08-01T09:00:00Z',
  },
  {
    id: 'w2',
    title: 'CCF Holding — IT',
    rate: 280,
    has_draft: true,
    checks_pass: 6,
    checks_total: 6,
    sent_at: '2026-07-31T10:00:00Z',
    updated_at: '2026-07-31T10:00:00Z',
  },
];

const ARCHIVES = [
  {
    id: 'w9',
    title: 'Vieux dossier',
    rate: 0,
    has_draft: false,
    checks_pass: 0,
    checks_total: 0,
    archived_at: '2026-07-01T10:00:00Z',
    updated_at: '2026-07-01T10:00:00Z',
  },
];

function mockFetch() {
  const calls: Array<[string, RequestInit | undefined]> = [];
  global.fetch = jest.fn((url: string, opts?: RequestInit) => {
    calls.push([String(url), opts]);
    const u = String(url);
    if (u.includes('archived=true')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ drafts: ARCHIVES }) });
    }
    if (u.includes('/admin/compose/drafts')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ drafts: ACTIFS }) });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
  }) as unknown as typeof fetch;
  return calls;
}

function setup(props: Partial<React.ComponentProps<typeof WorkshopSwitcher>> = {}) {
  const onOpen = jest.fn();
  const onCurrentChanged = jest.fn();
  render(
    <WorkshopSwitcher
      currentId="w1"
      currentTitle="Galadrim — AI agents"
      onOpen={onOpen}
      onCurrentChanged={onCurrentChanged}
      {...props}
    />,
  );
  return { onOpen, onCurrentChanged };
}

describe('WorkshopSwitcher', () => {
  beforeEach(() => {
    mockFetch();
  });

  // Le nom de l'atelier ouvert doit être lisible SANS ouvrir quoi que ce soit : c'est le manque
  // principal que cette fonctionnalité comble — on ne savait pas sur quelle annonce on travaillait.
  it('affiche le nom de l’atelier courant en permanence', () => {
    setup();
    expect(screen.getByTestId('compose-workshop-button')).toHaveTextContent('Galadrim — AI agents');
  });

  // Un atelier tout juste créé n'a pas encore de nom (il arrive à la 1re vérification) : il faut
  // quand même afficher quelque chose de cliquable, pas un bouton vide.
  it('retombe sur un libellé traduit quand l’atelier n’a pas encore de nom', () => {
    setup({ currentTitle: '   ' });
    expect(screen.getByTestId('compose-workshop-button')).toHaveTextContent('untitled');
  });

  // La liste n'est PAS chargée au montage : on vient sur cet écran pour rédiger, pas pour consulter
  // l'historique. La charger à l'ouverture garantit aussi qu'elle est à jour à chaque consultation.
  it('ne charge la liste qu’à l’ouverture', async () => {
    const calls = mockFetch();
    setup();
    expect(calls).toHaveLength(0);

    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-list')).toBeInTheDocument());
    expect(calls.some(([u]) => u.includes('/admin/compose/drafts'))).toBe(true);
  });

  it('liste les ateliers et remonte la bascule au parent', async () => {
    const { onOpen } = setup();

    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-open-w2')).toBeInTheDocument());
    expect(screen.getByTestId('compose-workshop-open-w2')).toHaveTextContent('CCF Holding — IT');

    fireEvent.click(screen.getByTestId('compose-workshop-open-w2'));
    expect(onOpen).toHaveBeenCalledWith('w2');
    // La liste se referme : on a choisi, on retourne au travail.
    expect(screen.queryByTestId('compose-workshop-list')).not.toBeInTheDocument();
  });

  // Archives dans un onglet séparé — elles doivent rester atteignables sans encombrer la vue
  // courante, puisqu'on ne supprime jamais rien.
  it('bascule sur les archives', async () => {
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-open-w1')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-workshop-tab-archived'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-open-w9')).toBeInTheDocument());
    expect(screen.queryByTestId('compose-workshop-open-w1')).not.toBeInTheDocument();
  });

  it('archive via le menu de la ligne, sans jamais supprimer', async () => {
    const calls = mockFetch();
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-menu-w2')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-workshop-menu-w2'));
    fireEvent.click(screen.getByTestId('compose-workshop-archive-w2'));

    await waitFor(() => {
      const patch = calls.find(([u, o]) => u.includes('/drafts/w2/meta') && o?.method === 'PATCH');
      expect(patch).toBeDefined();
      expect(JSON.parse(String(patch![1]!.body))).toEqual({ archived: true });
    });
    // AUCUNE suppression n'est émise : la seule sortie est l'archivage.
    expect(calls.some(([, o]) => o?.method === 'DELETE')).toBe(false);
  });

  // Un atelier archivé se réactive — c'est ce qui distingue l'archivage d'une suppression.
  it('réactive un atelier archivé', async () => {
    const calls = mockFetch();
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-tab-archived')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('compose-workshop-tab-archived'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-menu-w9')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-workshop-menu-w9'));
    fireEvent.click(screen.getByTestId('compose-workshop-archive-w9'));

    await waitFor(() => {
      const patch = calls.find(([u, o]) => u.includes('/drafts/w9/meta') && o?.method === 'PATCH');
      expect(JSON.parse(String(patch![1]!.body))).toEqual({ archived: false });
    });
  });

  it('marque un atelier comme envoyé, et sait revenir en arrière', async () => {
    const calls = mockFetch();
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-menu-w1')).toBeInTheDocument());

    // w1 n'est pas envoyé → on le marque.
    fireEvent.click(screen.getByTestId('compose-workshop-menu-w1'));
    fireEvent.click(screen.getByTestId('compose-workshop-sent-w1'));
    await waitFor(() => {
      const patch = calls.find(([u, o]) => u.includes('/drafts/w1/meta') && o?.method === 'PATCH');
      expect(JSON.parse(String(patch![1]!.body))).toEqual({ sent: true });
    });

    // w2 l'est déjà (sent_at présent) → l'action doit l'annuler, pas le remarquer.
    fireEvent.click(screen.getByTestId('compose-workshop-menu-w2'));
    fireEvent.click(screen.getByTestId('compose-workshop-sent-w2'));
    await waitFor(() => {
      const patch = calls.find(([u, o]) => u.includes('/drafts/w2/meta') && o?.method === 'PATCH');
      expect(JSON.parse(String(patch![1]!.body))).toEqual({ sent: false });
    });
  });

  // Renommage EN PLACE : elle voit la ligne qu'elle modifie au milieu des autres, donc elle peut
  // choisir un nom qui s'en distingue.
  it('renomme en place et n’envoie que le titre', async () => {
    const calls = mockFetch();
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-menu-w1')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-workshop-menu-w1'));
    fireEvent.click(screen.getByTestId('compose-workshop-rename-w1'));

    const input = await screen.findByTestId('compose-workshop-rename-input-w1');
    fireEvent.change(input, { target: { value: 'Mon nom à moi' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() => {
      const patch = calls.find(([u, o]) => u.includes('/drafts/w1/meta') && o?.method === 'PATCH');
      expect(JSON.parse(String(patch![1]!.body))).toEqual({ title: 'Mon nom à moi' });
    });
  });

  // Renommer l'atelier OUVERT doit remonter au parent : sinon le titre en tête d'écran
  // contredirait la liste juste en dessous.
  it('prévient le parent quand l’atelier courant change', async () => {
    const { onCurrentChanged } = setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-menu-w1')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-workshop-menu-w1'));
    fireEvent.click(screen.getByTestId('compose-workshop-rename-w1'));
    const input = await screen.findByTestId('compose-workshop-rename-input-w1');
    fireEvent.change(input, { target: { value: 'Renommé' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() =>
      expect(onCurrentChanged).toHaveBeenCalledWith(expect.objectContaining({ title: 'Renommé' })),
    );
  });

  // Échap ferme : un panneau flottant qu'on ne peut refermer qu'en retrouvant son bouton est un piège.
  it('se ferme avec Échap', async () => {
    setup();
    fireEvent.click(screen.getByTestId('compose-workshop-button'));
    await waitFor(() => expect(screen.getByTestId('compose-workshop-list')).toBeInTheDocument());

    fireEvent.keyDown(document, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByTestId('compose-workshop-list')).not.toBeInTheDocument());
  });
});
