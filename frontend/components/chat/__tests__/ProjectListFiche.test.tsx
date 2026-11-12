import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ProjectListFiche } from '../ProjectListFiche';

// Avant ce fix, les cartes n'étaient pas cliquables — impossible d'ouvrir la fiche détail d'un
// projet sans retaper une question au chat. BlogListFiche a déjà ce pattern (carte = lien).

const projects = [
  { Name: 'maicivy', Title: 'maicivy', Category: 'Web', ShortDesc: 'Un CV interactif.', TechStack: ['Go'] },
  { Name: 'drifterra', Title: 'Drifterra', Category: 'Jeu', ShortDesc: 'Un jeu de nav.', TechStack: ['Rust'] },
];

describe('ProjectListFiche — cartes cliquables', () => {
  it('envoie un message d\'ouverture localisé au clic sur une carte', async () => {
    const onProjectClick = jest.fn();
    render(<ProjectListFiche data={projects} onProjectClick={onProjectClick} />);

    await userEvent.click(screen.getByRole('button', { name: /maicivy/ }));

    expect(onProjectClick).toHaveBeenCalledTimes(1);
    expect(onProjectClick).toHaveBeenCalledWith('Montre-moi le projet maicivy');
  });

  it('chaque carte envoie le message correspondant À SON PROPRE projet', async () => {
    const onProjectClick = jest.fn();
    render(<ProjectListFiche data={projects} onProjectClick={onProjectClick} />);

    await userEvent.click(screen.getByRole('button', { name: /Drifterra/ }));

    expect(onProjectClick).toHaveBeenCalledWith('Montre-moi le projet Drifterra');
  });
});
