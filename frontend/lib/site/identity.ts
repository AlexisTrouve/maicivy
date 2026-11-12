/**
 * identity.ts — l'identité publique d'Alexis, données STRUCTURÉES (pas du texte d'interface).
 *
 * QUOI : le titre professionnel et les profils officiels, lus par le JSON-LD Person de l'accueil et
 * par les liens du pied de page.
 * POURQUOI des constantes et pas next-intl : le JSON-LD est lu par Google pour construire l'entité
 * « Alexis Trouvé » ; il doit être stable d'une langue à l'autre. Le positionnement (« Agent
 * Orchestrator — Technomancer ») est une signature gardée en anglais dans les 5 langues, comme
 * `montagne.voice.tagline`. À tenir synchrone avec maiProFiles /profile.
 */

export const JOB_TITLE = 'Agent Orchestrator — Technomancer';

export const LINKEDIN_URL = 'https://www.linkedin.com/in/alexis-trouve-432397a9/';
export const GITHUB_URL = 'https://github.com/AlexisTrouve';

/** Person schema.org — rendu dans le HTML serveur de l'accueil. */
export const PERSON_SCHEMA = {
  '@context': 'https://schema.org',
  '@type': 'Person',
  name: 'Alexis Trouvé',
  url: 'https://maicivy.etheryale.com',
  jobTitle: JOB_TITLE,
  address: { '@type': 'PostalAddress', addressCountry: 'France' },
  sameAs: [LINKEDIN_URL, GITHUB_URL],
} as const;
