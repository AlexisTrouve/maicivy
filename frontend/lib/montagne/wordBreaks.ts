/**
 * wordBreaks.ts — où un nom de projet peut passer à la ligne sans être coupé n'importe où.
 *
 * POURQUOI : sous une tuile de téléphone, le nom tient dans ~54 px. Beaucoup de noms sont UN
 * seul mot en camelCase (« GroveEngine », « ProjectMind », « twitterScraper ») ; sans point de
 * coupure, le navigateur les tranche à la lettre près (mesuré : « GroveEngi / ne »). On autorise
 * donc la coupure aux frontières que l'œil lit déjà : minuscule → Majuscule, et après « _ » ou « - ».
 * COMMENT : rend les segments du nom ; le composant les sépare par <wbr> (coupure permise, aucun
 * caractère ajouté). Un nom sans frontière reste un seul segment.
 */
export function nameSegments(name: string): string[] {
  return name.split(/(?<=[a-zà-ÿ])(?=[A-Z])|(?<=[_-])(?=\S)/u).filter((s) => s.length > 0);
}
