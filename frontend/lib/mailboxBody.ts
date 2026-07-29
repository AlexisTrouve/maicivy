// Découpe le corps d'un mail mailbox en segments texte/lien.
//
// POURQUOI : les mails Malt/HubSpot embarquent des URLs de tracking énormes (des centaines de
// caractères, ex: "Malt (https://d2Z2Gr04.eu1.hubspotlinks.com/Ctc/W3+113/...)") directement dans le
// texte brut — pas du HTML, du vrai text/plain, mais illisible tel quel dans un <pre>. On garde le
// body_text intact en base (aucune perte de donnée) et on ne transforme QUE l'affichage : chaque URL
// devient un segment "link" distinct, à rendre comme un petit lien cliquable plutôt que la chaîne
// brute complète.
export type MailboxBodySegment = { type: 'text'; value: string } | { type: 'link'; value: string };

const URL_RE = /(https?:\/\/[^\s)]+)/g;

export function splitMailboxBody(text: string): MailboxBodySegment[] {
  const segments: MailboxBodySegment[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  URL_RE.lastIndex = 0; // regex global partagée par module — reset requis avant chaque parcours
  while ((match = URL_RE.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ type: 'text', value: text.slice(lastIndex, match.index) });
    }
    segments.push({ type: 'link', value: match[0] });
    lastIndex = match.index + match[0].length;
  }
  if (lastIndex < text.length) {
    segments.push({ type: 'text', value: text.slice(lastIndex) });
  }
  return segments;
}

// shortLinkLabel retourne le domaine d'une URL pour l'affichage (ex: "hubspotlinks.com" plutôt que
// la chaîne de tracking complète) — donne une idée de la destination sans le bruit visuel.
// URL invalide (ne devrait pas arriver, la regex de splitMailboxBody ne matche que du http(s)://
// bien formé) → repli sur "lien" plutôt que planter le rendu.
export function shortLinkLabel(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '');
  } catch {
    return 'lien';
  }
}
