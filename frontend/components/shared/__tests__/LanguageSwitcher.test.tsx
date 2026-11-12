import { fireEvent, render, screen } from '@testing-library/react';
import { useParams } from 'next/navigation';

import { LanguageSwitcher } from '../LanguageSwitcher';

/**
 * LanguageSwitcher — partagé par l'en-tête du site (Header) et son pied de page (Footer).
 *
 * Verrou du défaut « FR FR » (montagne v4) : le bouton affichait le drapeau emoji PUIS le code.
 * Un drapeau emoji est une paire de « regional indicators » ; Windows n'a pas de glyphe pour eux,
 * Chromium y dessine donc les deux lettres — « 🇫🇷 » devient « FR » à l'écran, d'où « FR FR ».
 * Le bouton ne porte plus que le code ; le menu, lui, garde le nom de chaque langue — sans drapeau
 * non plus (même défaut : « FR Français » sous Windows).
 */

// Surcharge locale du mock global de next/navigation : la locale varie selon le cas.
jest.mock('next/navigation', () => ({
  useParams: jest.fn(() => ({ locale: 'fr' })),
  usePathname: () => '/fr/portfolio',
}));

const replace = jest.fn();
jest.mock('@/i18n/navigation', () => ({
  useRouter: () => ({ replace }),
}));

/** Plage Unicode des « regional indicators » (lettres de drapeau). */
const REGIONAL_INDICATOR = /[\u{1F1E6}-\u{1F1FF}]/u;

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    replace.mockClear();
    (useParams as jest.Mock).mockReturnValue({ locale: 'fr' });
  });

  it('le bouton affiche le code seul — aucun drapeau qui se lirait comme un second code', () => {
    render(<LanguageSwitcher />);
    const trigger = screen.getByRole('button', { expanded: false });
    expect(trigger.textContent?.replace(/\s+/g, '')).toBe('FR');
    expect(trigger.textContent).not.toMatch(REGIONAL_INDICATOR);
  });

  it("nomme le bouton pour le lecteur d'écran (i18n), le code restant le seul texte visible", () => {
    render(<LanguageSwitcher />);
    expect(screen.getByRole('button', { name: 'Langue : Français' })).toBeInTheDocument();
  });

  it("s'ouvre vers le haut dans le pied de page", () => {
    render(<LanguageSwitcher placement="up" />);
    fireEvent.click(screen.getByRole('button', { expanded: false }));
    expect(screen.getByRole('listbox')).toHaveClass('bottom-full');
  });

  it('suit la locale courante', () => {
    (useParams as jest.Mock).mockReturnValue({ locale: 'zh' });
    render(<LanguageSwitcher />);
    expect(screen.getByRole('button', { expanded: false }).textContent?.replace(/\s+/g, '')).toBe('ZH');
  });

  it('le menu liste les cinq langues et bascule la même page dans la langue choisie', () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByRole('button', { expanded: false }));
    const options = screen.getAllByRole('option');
    expect(options).toHaveLength(5);
    for (const o of options) expect(o.textContent).not.toMatch(REGIONAL_INDICATOR);
    expect(screen.getByRole('option', { selected: true })).toHaveTextContent('Français');
    fireEvent.click(screen.getByRole('option', { name: /English/ }));
    expect(replace).toHaveBeenCalledWith('/portfolio', { locale: 'en' });
  });
});
