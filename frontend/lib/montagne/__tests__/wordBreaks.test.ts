import { nameSegments } from '../wordBreaks';

/** Points de coupure des noms sous les tuiles (wordBreaks.ts). */
describe('nameSegments', () => {
  it('coupe aux frontières camelCase', () => {
    expect(nameSegments('GroveEngine')).toEqual(['Grove', 'Engine']);
    expect(nameSegments('twitterScraper')).toEqual(['twitter', 'Scraper']);
    expect(nameSegments('LucerianeRAG')).toEqual(['Luceriane', 'RAG']);
  });

  it('coupe après « _ » et « - », jamais avant', () => {
    expect(nameSegments('etheryale_duo')).toEqual(['etheryale_', 'duo']);
    expect(nameSegments('wechat-homework')).toEqual(['wechat-', 'homework']);
  });

  it('laisse intacts les sigles et les noms sans frontière', () => {
    expect(nameSegments('AVEG')).toEqual(['AVEG']);
    expect(nameSegments('Eikon')).toEqual(['Eikon']);
    expect(nameSegments('Feed Generator')).toEqual(['Feed Generator']);
  });

  it('ne perd ni n’ajoute aucun caractère', () => {
    for (const n of ['MonkeyScraper', 'AISSIA (CELUNA)', 'DAOS (Dawi: Age of Stone)', 'Video to MP3 Transcriptor']) {
      expect(nameSegments(n).join('')).toBe(n);
    }
  });
});
