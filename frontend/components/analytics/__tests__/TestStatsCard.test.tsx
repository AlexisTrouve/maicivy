import { render, screen } from '@testing-library/react';
import TestStatsCard from '../TestStatsCard';
import stats from '@/lib/test-stats.json';

// next-intl + lucide via mocks GLOBAUX (jest.config). useTranslations('analytics.testStats') résout le FR.

describe('TestStatsCard', () => {
  it('affiche le titre + total + détail backend/frontend depuis test-stats.json', () => {
    render(<TestStatsCard />);

    expect(screen.getByText('Tests & Qualité')).toBeInTheDocument();
    expect(screen.getByTestId('test-stats-card')).toBeInTheDocument();
    // Chiffres réels lus dans le JSON (pas codés en dur). On compare les CHIFFRES seuls (insensible au
    // séparateur de milliers que toLocaleST ajoute dès 1000, ex "1 006" en fr) → robuste à l'ICU.
    const digitsEq = (n: number) => (content: string) => content.replace(/\D/g, '') === String(n);
    expect(screen.getByText(digitsEq(stats.total))).toBeInTheDocument();
    expect(screen.getByText(digitsEq(stats.backend.tests))).toBeInTheDocument();
    expect(screen.getByText(digitsEq(stats.frontend.tests))).toBeInTheDocument();
  });

  it('affiche le badge "100% verts" quand allGreen', () => {
    render(<TestStatsCard />);
    if (stats.allGreen) {
      expect(screen.getByText('100% verts')).toBeInTheDocument();
    }
  });
});
