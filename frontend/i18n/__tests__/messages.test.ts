import { loadMessages } from '../messages';

// Verrouille le loader statique de messages (remplaçant turbopack-safe de l'import dynamique).
describe('loadMessages', () => {
  it('charge les messages de la bonne locale', () => {
    expect(loadMessages('fr').nav.home).toBe('Accueil');
    expect(loadMessages('en').nav.home).toBe('Home');
    expect(loadMessages('de').nav.home).toBe('Startseite');
  });

  it('charge aussi it et zh (structure présente)', () => {
    expect(loadMessages('it').nav.home).toBeTruthy();
    expect(loadMessages('zh').nav.home).toBeTruthy();
  });

  it('fallback sur la locale par défaut (fr) si la locale est inconnue', () => {
    // Locale absente → on retombe sur l'objet FR (même référence que loadMessages('fr')).
    expect(loadMessages('xx')).toBe(loadMessages('fr'));
    expect(loadMessages('').nav.home).toBe('Accueil');
  });
});
