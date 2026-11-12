import { render, screen } from '@testing-library/react';
import { ProjectFiche } from '../ProjectFiche';

// Avant ce fix, ProjectFiche n'avait AUCUN lien vers le code/la démo (contrairement à BlogFiche, qui
// en a un depuis toujours) — un visiteur qui chatte sur un projet ne pouvait pas cliquer pour voir
// le repo/la démo réels. Ces tests verrouillent le rendu réel des liens quand ils sont fournis, et
// leur absence propre (pas de lien mort/vide) quand ils ne le sont pas.

const baseData = {
  Name: 'maicivy',
  Title: 'maicivy',
  Category: 'Web',
  ShortDesc: 'Un CV interactif.',
  KeyFeatures: [],
  TechStack: ['Go', 'Next.js'],
  Stats: [],
  SkillsTags: [],
};

describe('ProjectFiche — liens GitHub/démo', () => {
  it('affiche les deux liens quand GithubURL et DemoURL sont fournis', () => {
    render(<ProjectFiche data={{ ...baseData, GithubURL: 'https://github.com/example/maicivy', DemoURL: 'https://maicivy.etheryale.com' }} />);

    const github = screen.getByRole('link', { name: 'Voir le code' });
    expect(github).toHaveAttribute('href', 'https://github.com/example/maicivy');
    expect(github).toHaveAttribute('target', '_blank');

    const demo = screen.getByRole('link', { name: 'Voir la démo' });
    expect(demo).toHaveAttribute('href', 'https://maicivy.etheryale.com');
  });

  it("n'affiche aucun lien quand ni GithubURL ni DemoURL ne sont fournis (pas de lien mort)", () => {
    render(<ProjectFiche data={baseData} />);
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('affiche seulement le lien démo quand GithubURL est absent (ex: repo Gitea privé masqué)', () => {
    render(<ProjectFiche data={{ ...baseData, DemoURL: 'https://maicivy.etheryale.com' }} />);
    expect(screen.queryByRole('link', { name: 'Voir le code' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Voir la démo' })).toBeInTheDocument();
  });
});
