import React from 'react';
import { render, screen } from '@testing-library/react';
import { CVSkeleton } from '../CVSkeleton';

// Le squelette est vérifié sur sa STRUCTURE (data-testid) et sur les deux propriétés qui comptent :
// il ne porte aucune couleur propre (jetons du thème uniquement) et il est déterministe (rendu
// serveur : deux rendus doivent produire le même HTML). Plus de classes de mise en forme épinglées :
// la refonte les change, l'utilisateur ne les voit pas.
describe('CVSkeleton', () => {
  it('rend les trois sections du CV : expériences, compétences, projets', () => {
    render(<CVSkeleton />);
    expect(screen.getAllByTestId('cv-skeleton-section')).toHaveLength(3);
    expect(screen.getAllByTestId('cv-skeleton-experience')).toHaveLength(3);
    expect(screen.getAllByTestId('cv-skeleton-skill')).toHaveLength(8);
    expect(screen.getAllByTestId('cv-skeleton-project')).toHaveLength(6);
  });

  it('pulse et reste caché des lecteurs d’écran', () => {
    render(<CVSkeleton />);
    const root = screen.getByTestId('cv-skeleton');
    expect(root).toHaveClass('animate-pulse');
    expect(root).toHaveAttribute('aria-hidden', 'true');
  });

  it('dispose les projets dans la même grille que la page finale', () => {
    render(<CVSkeleton />);
    const grid = screen.getAllByTestId('cv-skeleton-project')[0].parentElement;
    expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3');
  });

  it("n'utilise que les jetons du thème — aucune couleur Tailwind en dur", () => {
    const { container } = render(<CVSkeleton />);
    expect(container.innerHTML).not.toMatch(/\b(?:bg|text|border)-(?:gray|slate|blue|purple|white|black)\b/);
  });

  it('est déterministe : deux rendus donnent le même HTML (pas de Math.random)', () => {
    const a = render(<CVSkeleton />).container.innerHTML;
    const b = render(<CVSkeleton />).container.innerHTML;
    expect(a).toBe(b);
  });
});
