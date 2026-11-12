import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LeftPanel } from '../LeftPanel';

// Priorité d'affichage du LeftPanel : tips > relances contextuelles (suggest_followups, LLM) > hints
// statiques. Avant ce fix, il n'y avait que 2 niveaux (tips > hints) — les relances contextuelles
// n'existaient pas, le pool restait un tirage aléatoire jamais influencé par la conversation.

describe('LeftPanel — priorité tips > relances > hints', () => {
  it('affiche les relances quand il n\'y a pas de tips', () => {
    render(
      <LeftPanel
        tips={[]}
        onTipClose={jest.fn()}
        onHintClick={jest.fn()}
        followups={['Quels sont tes projets en Rust ?', 'Et en Go ?']}
      />,
    );
    expect(screen.getByText('Quels sont tes projets en Rust ?')).toBeInTheDocument();
    expect(screen.getByText('Et en Go ?')).toBeInTheDocument();
    expect(screen.getByText('Pour continuer')).toBeInTheDocument();
  });

  it('les tips restent PRIORITAIRES sur les relances (même si les deux sont présents)', () => {
    render(
      <LeftPanel
        tips={[{ id: 't1', text: 'Un tip important' }]}
        onTipClose={jest.fn()}
        onHintClick={jest.fn()}
        followups={['Une relance qui ne devrait PAS s\'afficher']}
      />,
    );
    expect(screen.getByText('Un tip important')).toBeInTheDocument();
    expect(screen.queryByText('Une relance qui ne devrait PAS s\'afficher')).not.toBeInTheDocument();
  });

  it('cliquer une relance appelle onHintClick avec le texte exact de la question', async () => {
    const onHintClick = jest.fn();
    render(
      <LeftPanel
        tips={[]}
        onTipClose={jest.fn()}
        onHintClick={onHintClick}
        followups={['Quels sont tes projets en Rust ?']}
      />,
    );
    await userEvent.click(screen.getByText('Quels sont tes projets en Rust ?'));
    expect(onHintClick).toHaveBeenCalledWith('Quels sont tes projets en Rust ?');
  });

  it('sans tips ni relances, retombe sur les hints statiques (comportement existant préservé)', () => {
    render(<LeftPanel tips={[]} onTipClose={jest.fn()} onHintClick={jest.fn()} followups={[]} />);
    expect(screen.getByText('Questions')).toBeInTheDocument();
    expect(screen.queryByText('Pour continuer')).not.toBeInTheDocument();
  });
});
