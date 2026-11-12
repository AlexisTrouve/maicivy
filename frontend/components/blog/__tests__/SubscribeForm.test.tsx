// SubscribeForm — abonnement par email, réseau SIMULÉ (aucun vrai abonnement n'est envoyé).
//
// QUOI : les sujets sont des interrupteurs dont l'état se lit au glyphe ✓ (pas à la seule couleur),
// l'envoi transmet exactement l'email et les sujets cochés, et les deux issues (succès, échec) se
// disent avec un glyphe et un rôle ARIA. Le bouton change de libellé pendant l'envoi (plus de « … »
// en dur).
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

const getTopics = jest.fn();
const subscribe = jest.fn();
jest.mock('@/lib/api', () => ({
  blogApi: {
    getTopics: (...a: unknown[]) => getTopics(...a),
    subscribe: (...a: unknown[]) => subscribe(...a),
  },
}));

import { SubscribeForm } from '../SubscribeForm';

describe('SubscribeForm', () => {
  beforeEach(() => {
    getTopics.mockReset().mockResolvedValue({ topics: ['Drifterra', 'tech'] });
    subscribe.mockReset();
  });

  it('un sujet coché porte aria-pressed et le glyphe ✓', async () => {
    render(<SubscribeForm />);
    const sujet = await screen.findByRole('button', { name: 'Drifterra (le jeu)' });
    expect(sujet).toHaveAttribute('aria-pressed', 'false');

    fireEvent.click(sujet);
    expect(sujet).toHaveAttribute('aria-pressed', 'true');
    expect(sujet).toHaveTextContent('✓');
  });

  it('envoie l’email et les sujets cochés, puis annonce le succès', async () => {
    let resoudre: () => void = () => {};
    subscribe.mockReturnValue(new Promise<void>((r) => (resoudre = r)));
    render(<SubscribeForm />);

    fireEvent.click(await screen.findByRole('button', { name: 'Dev / tech' }));
    fireEvent.change(screen.getByLabelText('Adresse email'), { target: { value: '  a@b.co ' } });
    fireEvent.click(screen.getByRole('button', { name: "S'abonner" }));

    // Pendant l'envoi : libellé traduit, bouton désactivé.
    expect(await screen.findByRole('button', { name: 'Envoi…' })).toBeDisabled();
    expect(subscribe).toHaveBeenCalledWith('a@b.co', ['tech']);

    resoudre();
    expect(await screen.findByRole('status')).toHaveTextContent('Inscrit.');
  });

  it('un échec s’annonce (role=alert, glyphe ✕) et laisse le formulaire en place', async () => {
    subscribe.mockRejectedValue(new Error('400'));
    render(<SubscribeForm />);

    fireEvent.change(screen.getByLabelText('Adresse email'), { target: { value: 'a@b.co' } });
    fireEvent.click(screen.getByRole('button', { name: "S'abonner" }));

    const alerte = await screen.findByRole('alert');
    expect(alerte).toHaveTextContent('✕');
    expect(alerte).toHaveTextContent('réessaie');
    await waitFor(() => expect(screen.getByRole('button', { name: "S'abonner" })).toBeEnabled());
  });
});
