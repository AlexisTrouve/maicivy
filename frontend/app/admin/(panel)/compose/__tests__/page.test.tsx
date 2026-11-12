import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminComposeTool from '../page';
import { streamChat } from '@/lib/chat-api';
import * as chatImages from '@/lib/chat-images';

// QUOI : vérifie le contrat de l'écran compose côté client.
//
// POURQUOI ces cas précis : l'opérateur ne lit pas la langue du message. Trois garanties le protègent
// et sont vérifiées ici — (1) le draft ne peut PAS être édité à la main (sinon il contournerait le
// vérificateur et la traduction), (2) le draft affiché vient de l'INPUT du tool_call et non d'un texte
// extrait de la réponse de l'agent, (3) la vérification est demandée après chaque tour.
//
// COMMENT : streamChat est mocké pour piloter les callbacks à la main (tool_call puis done), ce qui
// permet de simuler l'agent sans réseau.

jest.mock('@/lib/chat-api', () => ({ streamChat: jest.fn() }));

// fileToChatImage passe par FileReader + canvas, absents de jsdom. On ne remplace QUE lui : les
// fonctions pures (types acceptés, extraction depuis le presse-papier) restent les vraies, et sont
// couvertes par leurs propres tests unitaires.
jest.mock('@/lib/chat-images', () => ({
  ...jest.requireActual('@/lib/chat-images'),
  fileToChatImage: jest.fn(),
}));

const mockedStream = streamChat as jest.MockedFunction<typeof streamChat>;

const DRAFT = 'Bonjour,\n\nJe travaille exclusivement en remote.\n\nAlexis';

// driveAgent : rejoue un tour d'agent qui appelle set_draft puis termine.
function driveAgent(extra?: (cb: Parameters<typeof streamChat>[2]) => void) {
  mockedStream.mockImplementation(async (_message, _history, callbacks) => {
    callbacks.onToolCall('set_draft', { message: DRAFT });
    extra?.(callbacks);
    callbacks.onDone();
  });
}

// findCall : retrouve un appel fetch par son URL. POURQUOI plutôt qu'un index : l'écran émet aussi un
// GET de restauration au montage et un POST de sauvegarde en fin de tour — se fier à l'ordre rendrait
// ces tests cassants à chaque ajout d'appel, sans que le comportement testé ait changé.
function findCall(method: string, urlFragment: string) {
  return (global.fetch as jest.Mock).mock.calls.find(
    ([url, opts]) =>
      String(url).includes(urlFragment) && (opts?.method ?? 'GET').toUpperCase() === method,
  );
}

function startWithAnnounce() {
  fireEvent.change(screen.getByTestId('compose-announce'), {
    target: { value: 'Recherche freelance IA, mission longue.' },
  });
  fireEvent.click(screen.getByTestId('compose-start'));
}

describe('AdminComposeTool', () => {
  beforeEach(() => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          translation: 'Hello,\n\nI work exclusively remote.\n\nAlexis',
          mechanical: [
            { id: 'remote', status: 'pass' },
            { id: 'no_markdown', status: 'fail', detail: '**' },
          ],
          semantic: [{ id: 'terraform', label: 'Mentions Terraform', status: 'fail' }],
        }),
    });
    Object.assign(navigator, { clipboard: { writeText: jest.fn().mockResolvedValue(undefined) } });
  });

  afterEach(() => jest.restoreAllMocks());

  // Le drapeau compose est ce qui débloque les tools de rédaction côté backend (qui ne les accorde
  // qu'à un owner). S'il n'est pas envoyé, l'agent ne peut pas écrire dans le draft.
  it('demande les capacités compose au backend', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(mockedStream).toHaveBeenCalled());
    // On vérifie la capacité demandée, pas la forme exacte de l'objet : y ajouter un champ ne doit
    // pas casser un test qui porte sur le drapeau compose.
    expect(mockedStream.mock.calls[0][4]).toMatchObject({ compose: true });
  });

  it('affiche le draft poussé par le tool_call set_draft', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(screen.getByTestId('compose-draft')).toHaveTextContent('exclusivement en remote'));
  });

  // Garantie centrale : le livrable est en lecture seule. Une édition manuelle contournerait la
  // checklist et la traduction, et l'opérateur ne peut pas relire ce qu'il aurait modifié.
  it('rend le draft et la traduction non éditables', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(screen.getByTestId('compose-draft')).toBeInTheDocument());
    for (const id of ['compose-draft', 'compose-translation']) {
      const node = screen.getByTestId(id);
      expect(node.tagName).not.toBe('TEXTAREA');
      expect(node.tagName).not.toBe('INPUT');
      expect(node).not.toHaveAttribute('contenteditable', 'true');
    }
  });

  it('demande la vérification après le tour et affiche traduction + checklist', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(findCall('POST', '/admin/compose/verify')).toBeDefined());
    const [url, opts] = findCall('POST', '/admin/compose/verify')!;
    expect(String(url)).toMatch(/\/api\/v1\/admin\/compose\/verify$/);
    expect(opts.credentials).toBe('include');
    const body = JSON.parse(opts.body);
    expect(body.draft).toContain('exclusivement en remote');
    expect(body.announce).toContain('Recherche freelance IA');

    await waitFor(() =>
      expect(screen.getByTestId('compose-translation')).toHaveTextContent('I work exclusively remote'),
    );
    expect(screen.getByTestId('check-remote')).toHaveAttribute('data-status', 'pass');
    expect(screen.getByTestId('check-no_markdown')).toHaveAttribute('data-status', 'fail');
    expect(screen.getByTestId('check-semantic-terraform')).toHaveAttribute('data-status', 'fail');
  });

  // Le TJM arrive par un tool, pas par une saisie : un seul chemin de mutation, donc la checklist ne
  // peut pas se désynchroniser de la valeur retenue.
  it('reprend le TJM fixé par set_rate et le renvoie à la vérification', async () => {
    driveAgent((cb) => cb.onToolCall('set_rate', { amount: 300, rationale: 'stack agentique' }));
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(findCall('POST', '/admin/compose/verify')).toBeDefined());
    const body = JSON.parse(findCall('POST', '/admin/compose/verify')![1].body);
    expect(body.retained_rate).toBe(300);
  });

  // Les tool calls sont visibles : sans ça le message change sous les yeux de l'opérateur sans
  // explication.
  it('affiche les appels de tools de l’agent', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(screen.getByTestId('compose-toolcall-set_draft')).toBeInTheDocument());
  });

  // Sans persistance, un rafraîchissement perd le message en cours — et l'opérateur, qui ne relit pas
  // le français, ne peut pas reconstituer plusieurs tours d'ajustement.
  it('sauvegarde l’atelier complet après le tour', async () => {
    driveAgent((cb) => cb.onToolCall('set_rate', { amount: 300, rationale: 'stack agentique' }));
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(findCall('POST', '/admin/compose/drafts')).toBeDefined());
    const body = JSON.parse(findCall('POST', '/admin/compose/drafts')![1].body);

    expect(body.draft).toContain('exclusivement en remote');
    expect(body.announce).toContain('Recherche freelance IA');
    expect(body.rate).toBe(300);
    expect(body.rate_rationale).toBe('stack agentique');
    // Le fil ET la checklist partent avec : au rechargement, tout doit décrire la MÊME version.
    expect(body.messages.length).toBeGreaterThan(0);
    expect(body.checks.mechanical.length).toBeGreaterThan(0);
    expect(body.translation).toContain('I work exclusively remote');
  });

  // La restauration se fait en DEUX temps depuis l'ajout de l'historique : la liste (allégée, sans
  // le contenu) donne l'identifiant du plus récent, puis on charge CE contenu-là.
  //
  // POURQUOI la liste ne porte plus le contenu : le fil de conversation embarque les captures
  // d'écran en base64. Mesuré en production — 3 ateliers pesaient 468 kB, dont un seul à 455 kB. La
  // liste complète partait à chaque ouverture de l'écran pour n'en restaurer qu'un, et garder
  // l'historique aggravait ce coût à chaque atelier supplémentaire.
  it('restaure le dernier atelier au montage, sans charger toute la liste', async () => {
    const complet = {
      id: 'draft-1',
      title: 'Galadrim — AI agents',
      announce: 'Ancienne annonce',
      draft: 'Message restauré',
      translation: 'Restored message',
      rate: 350,
      rate_rationale: 'mission longue',
      messages: [{ role: 'user', content: 'salut' }],
      checks: { mechanical: [{ id: 'remote', status: 'pass' }], semantic: [] },
    };

    (global.fetch as jest.Mock).mockImplementation((url: string, opts?: { method?: string }) => {
      const u = String(url);
      if ((opts?.method ?? 'GET') !== 'GET') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
      }
      // Atelier à l'unité — c'est ici, et seulement ici, que le contenu circule.
      if (u.includes('/admin/compose/drafts/draft-1')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(complet) });
      }
      // Liste : uniquement de quoi choisir.
      if (u.includes('/admin/compose/drafts')) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              drafts: [
                {
                  id: 'draft-1',
                  title: 'Galadrim — AI agents',
                  rate: 350,
                  has_draft: true,
                  checks_pass: 1,
                  checks_total: 1,
                  updated_at: '2026-08-01T09:00:00Z',
                },
              ],
            }),
        });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    });

    render(<AdminComposeTool />);

    await waitFor(() => expect(screen.getByTestId('compose-draft')).toHaveTextContent('Message restauré'));
    expect(screen.getByTestId('compose-translation')).toHaveTextContent('Restored message');
    expect(screen.getByTestId('compose-rate')).toHaveTextContent('350');
    expect(screen.getByTestId('check-remote')).toHaveAttribute('data-status', 'pass');
    // Le nom de l'atelier est affiché en tête : c'est le repère qui manquait le plus, on ne savait
    // pas sur quelle annonce on travaillait.
    expect(screen.getByTestId('compose-workshop-button')).toHaveTextContent('Galadrim — AI agents');
  });

  // Sans cette sortie, la restauration au montage ramènerait éternellement le même atelier et
  // l'opérateur ne pourrait jamais traiter une deuxième annonce. Régression attrapée par l'E2E.
  it('permet de repartir sur un atelier vierge', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();
    await waitFor(() => expect(screen.getByTestId('compose-draft')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('compose-new'));

    // Le formulaire de saisie revient, vide, et l'atelier précédent a disparu de l'écran.
    expect(screen.getByTestId('compose-announce')).toHaveValue('');
    expect(screen.queryByTestId('compose-draft')).not.toBeInTheDocument();
    expect(screen.queryByTestId('compose-rate')).not.toBeInTheDocument();

    // Et le tour suivant crée bien une NOUVELLE ligne (POST) au lieu d'écraser la précédente (PUT).
    (global.fetch as jest.Mock).mockClear();
    startWithAnnounce();
    await waitFor(() => expect(findCall('POST', '/admin/compose/drafts')).toBeDefined());
    expect(findCall('PUT', '/admin/compose/drafts')).toBeUndefined();
  });

  // L'opérateur ne lit pas le français : voir le RAISONNEMENT de l'agent et ce qu'il a réellement
  // écrit dans ses tools est sa seule fenêtre sur la qualité du travail avant l'envoi au client.
  it('affiche le raisonnement et les tools AVANT le message', async () => {
    mockedStream.mockImplementation(async (_message, _history, cb) => {
      cb.onThinking?.('Leur vrai problème est la mise en production.');
      cb.onToolCall('set_draft', { message: DRAFT });
      cb.onText('Voici une première version.');
      cb.onDone();
    });
    render(<AdminComposeTool />);
    startWithAnnounce();

    await waitFor(() => expect(screen.getByTestId('compose-toolcall-set_draft')).toBeInTheDocument());

    // Le raisonnement est rendu, pas seulement l'output.
    expect(screen.getByText(/mise en production/)).toBeInTheDocument();

    // ORDRE DOM : le tool précède le message auquel il a mené.
    const tool = screen.getByTestId('compose-toolcall-set_draft');
    const message = screen.getByText('Voici une première version.');
    expect(tool.compareDocumentPosition(message) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    // Le contenu du tool est consultable : savoir qu'il a appelé set_draft n'apprend rien, voir CE
    // qu'il a écrit explique pourquoi le message affiché a changé.
    expect(tool).toHaveTextContent('exclusivement en remote');
  });

  // Repliés par défaut : dix tours de raisonnement déplié rendraient la conversation illisible.
  it('replie le raisonnement des tours passés', async () => {
    mockedStream.mockImplementation(async (_message, _history, cb) => {
      cb.onThinking?.('Un raisonnement.');
      cb.onToolCall('set_draft', { message: DRAFT });
      cb.onDone();
    });
    render(<AdminComposeTool />);
    startWithAnnounce();

    // Regex plutôt qu'un indice figé : l'indice dépend du nombre de tours précédents, ce que ce test
    // n'a pas à connaître.
    await waitFor(() => expect(screen.getByTestId(/^compose-thinking-\d+$/)).toBeInTheDocument());
    expect(screen.getByTestId(/^compose-thinking-\d+$/)).not.toHaveAttribute('open');
  });

  // Captures d'écran : c'est ce qui permet à l'opératrice de faire répondre l'agent aux questions de
  // formulaire de la plateforme sans écrire à Alexi. Les trois gestes (dépôt, sélecteur, collage)
  // convergent vers le même point d'entrée — on teste ici celui qui est pilotable en jsdom.
  it('joint une capture et l’envoie avec le message', async () => {
    (chatImages.fileToChatImage as jest.Mock).mockResolvedValue({
      media_type: 'image/png',
      data: 'QUFBQQ==',
    });
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();
    await waitFor(() => expect(screen.getByTestId('compose-draft')).toBeInTheDocument());

    const fichier = new File(['x'], 'formulaire.png', { type: 'image/png' });
    fireEvent.change(screen.getByTestId('compose-file-input'), { target: { files: [fichier] } });

    // La vignette apparaît AVANT l'envoi : elle voit ce qu'elle s'apprête à joindre.
    await waitFor(() => expect(screen.getByTestId('compose-pending-images')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('compose-chat-input'), {
      target: { value: 'Que dois-je mettre dans ces champs ?' },
    });
    fireEvent.click(screen.getByTestId('compose-chat-send'));

    await waitFor(() => expect(mockedStream).toHaveBeenCalledTimes(2));
    expect(mockedStream.mock.calls[1][4]).toMatchObject({
      compose: true,
      images: [{ media_type: 'image/png', data: 'QUFBQQ==' }],
    });

    // La file est vidée : une capture ne doit pas se rattacher au message suivant.
    await waitFor(() => expect(screen.queryByTestId('compose-pending-images')).not.toBeInTheDocument());
  });

  // Un format refusé doit le DIRE. L'écarter en silence ferait répondre l'agent à côté sans que
  // personne comprenne pourquoi.
  it('signale un format d’image non supporté', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();
    await waitFor(() => expect(screen.getByTestId('compose-draft')).toBeInTheDocument());

    const mauvais = new File(['x'], 'doc.pdf', { type: 'application/pdf' });
    fireEvent.change(screen.getByTestId('compose-file-input'), { target: { files: [mauvais] } });

    await waitFor(() => expect(screen.getByTestId('compose-image-error')).toBeInTheDocument());
    expect(screen.queryByTestId('compose-pending-images')).not.toBeInTheDocument();
  });

  // Une capture SEULE doit partir : exiger du texte obligeait à écrire quelque chose d'inutile à
  // chaque envoi de capture.
  it('envoie une capture sans aucun texte', async () => {
    (chatImages.fileToChatImage as jest.Mock).mockResolvedValue({
      media_type: 'image/png',
      data: 'QUFBQQ==',
    });
    driveAgent();
    render(<AdminComposeTool />);

    // Dès le départ, sans brouillon : c'est le cas « capture de l'annonce ».
    const fichier = new File(['x'], 'annonce.png', { type: 'image/png' });
    fireEvent.change(screen.getByTestId('compose-file-input'), { target: { files: [fichier] } });
    await waitFor(() => expect(screen.getByTestId('compose-pending-images')).toBeInTheDocument());

    // Le bouton doit être actif alors que le champ texte est vide.
    expect(screen.getByTestId('compose-chat-send')).toBeEnabled();
    fireEvent.click(screen.getByTestId('compose-chat-send'));

    await waitFor(() => expect(mockedStream).toHaveBeenCalled());
    expect(mockedStream.mock.calls[0][4]).toMatchObject({
      images: [{ media_type: 'image/png', data: 'QUFBQQ==' }],
    });
  });

  // L'aperçu doit rester dans le fil : sans lui, après plusieurs tours on ne sait plus quelle réponse
  // répondait à quelle capture.
  it('affiche l’aperçu de la capture dans le fil', async () => {
    (chatImages.fileToChatImage as jest.Mock).mockResolvedValue({
      media_type: 'image/png',
      data: 'QUFBQQ==',
    });
    driveAgent();
    render(<AdminComposeTool />);

    fireEvent.change(screen.getByTestId('compose-file-input'), {
      target: { files: [new File(['x'], 'f.png', { type: 'image/png' })] },
    });
    await waitFor(() => expect(screen.getByTestId('compose-pending-images')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('compose-chat-send'));

    await waitFor(() => expect(screen.getByTestId('compose-msg-images-0')).toBeInTheDocument());
    const vignette = screen.getByTestId('compose-msg-images-0').querySelector('img');
    expect(vignette).toHaveAttribute('src', 'data:image/png;base64,QUFBQQ==');
  });

  // Un libellé fixe est indiscernable d'un écran gelé : l'indicateur doit être présent pendant TOUT
  // le tour, y compris une fois du texte déjà affiché (les temps morts sont entre les tools).
  it('montre l’activité de l’agent pendant tout le tour', async () => {
    let resoudre: (() => void) | undefined;
    mockedStream.mockImplementation(async (_m, _h, cb) => {
      cb.onText('Je commence…');
      await new Promise<void>((r) => {
        resoudre = r;
      });
      cb.onDone();
    });
    render(<AdminComposeTool />);
    startWithAnnounce();

    // Du texte est déjà affiché, et l'indicateur doit rester visible.
    await waitFor(() => expect(screen.getByTestId('compose-activity')).toBeInTheDocument());
    expect(screen.getByText('Je commence…')).toBeInTheDocument();

    resoudre?.();
    await waitFor(() => expect(screen.queryByTestId('compose-activity')).not.toBeInTheDocument());
  });

  // Une vérification ratée ne doit pas effacer la checklist précédente : une checklist vide se lirait
  // comme « rien à signaler », ce qui serait un mensonge.
  it('conserve la checklist si la vérification échoue', async () => {
    driveAgent();
    render(<AdminComposeTool />);
    startWithAnnounce();
    await waitFor(() => expect(screen.getByTestId('check-remote')).toBeInTheDocument());

    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('offline'));
    fireEvent.change(screen.getByTestId('compose-chat-input'), { target: { value: 'ajoute le remote' } });
    fireEvent.click(screen.getByTestId('compose-chat-send'));

    await waitFor(() => expect(mockedStream).toHaveBeenCalledTimes(2));
    expect(screen.getByTestId('check-remote')).toBeInTheDocument();
  });
});
