import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MessageBubble } from '../MessageBubble';

// navigator.clipboard n'existe pas en JSDOM par défaut.
Object.assign(navigator, { clipboard: { writeText: jest.fn().mockResolvedValue(undefined) } });

// Avant ce fix, un lien markdown "[GitHub](url)" ou une liste "- Go\n- React" s'affichait avec les
// crochets/tirets bruts VISIBLES (regex maison qui ne gérait que **gras**/`code`) — pas cassé, juste
// moche. Ces tests verrouillent le rendu RÉEL (vrai <a>, vraie <ul>/<li>), pas juste l'absence de crash.

describe('MessageBubble — rendu Markdown (assistant)', () => {
  it('rend un texte brut sans transformation', () => {
    render(<MessageBubble role="assistant" content="Bonjour, comment puis-je aider ?" />);
    expect(screen.getByText('Bonjour, comment puis-je aider ?')).toBeInTheDocument();
  });

  it('rend **gras** en <strong>', () => {
    render(<MessageBubble role="assistant" content="C'est **important**." />);
    const strong = screen.getByText('important');
    expect(strong.tagName).toBe('STRONG');
  });

  it('rend `code` en <code>', () => {
    render(<MessageBubble role="assistant" content="Lance `npm install`." />);
    const code = screen.getByText('npm install');
    expect(code.tagName).toBe('CODE');
  });

  it('rend un lien markdown en vrai <a>, PAS en texte brut avec crochets', () => {
    render(<MessageBubble role="assistant" content="Voir le projet sur [GitHub](https://github.com/example)." />);
    const link = screen.getByRole('link', { name: 'GitHub' });
    expect(link).toHaveAttribute('href', 'https://github.com/example');
    expect(link).toHaveAttribute('target', '_blank');
    // Le texte littéral avec crochets ne doit PLUS apparaître.
    expect(screen.queryByText(/\[GitHub\]/)).not.toBeInTheDocument();
  });

  it('rend une liste à puces en <ul><li>, PAS en lignes avec tirets bruts', () => {
    render(<MessageBubble role="assistant" content={'Mes langages préférés :\n- Go\n- TypeScript\n- Rust'} />);

    const list = screen.getByRole('list');
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(3);
    expect(items.map((li) => li.textContent)).toEqual(['Go', 'TypeScript', 'Rust']);
    // Les 3 puces consécutives forment UNE SEULE liste, pas trois listes séparées.
    expect(screen.getAllByRole('list')).toHaveLength(1);
    expect(list.tagName).toBe('UL');
  });

  it('gère gras/code/lien À L\'INTÉRIEUR d\'un item de liste', () => {
    render(<MessageBubble role="assistant" content={'- **Go** : voir [le repo](https://x.test)\n- `TypeScript`'} />);
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(2);
    expect(screen.getByText('Go').tagName).toBe('STRONG');
    expect(screen.getByRole('link', { name: 'le repo' })).toHaveAttribute('href', 'https://x.test');
    expect(screen.getByText('TypeScript').tagName).toBe('CODE');
  });
});

describe('MessageBubble — messages user : PAS de parsing markdown', () => {
  it('affiche le contenu utilisateur tel quel, même avec des caractères markdown', () => {
    render(<MessageBubble role="user" content="J'aime **ce** projet [test](url)" />);
    expect(screen.getByText("J'aime **ce** projet [test](url)")).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });
});

describe('MessageBubble — bouton copier', () => {
  it('copie le texte BRUT de la réponse assistant au clic, et affiche une confirmation', async () => {
    render(<MessageBubble role="assistant" content="Réponse **importante**." />);

    const button = screen.getByRole('button', { name: 'Copier' });
    await userEvent.click(button);

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('Réponse **importante**.');
    expect(screen.getByRole('button', { name: 'Copié !' })).toBeInTheDocument();
  });

  it("n'affiche PAS de bouton copier pour un message user", () => {
    render(<MessageBubble role="user" content="Ma question" />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it("n'affiche PAS de bouton copier pendant le streaming (texte encore partiel)", () => {
    render(<MessageBubble role="assistant" content="En cours..." isStreaming />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});
