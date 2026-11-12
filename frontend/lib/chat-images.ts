// Préparation des captures d'écran jointes à l'agent compose.
//
// POURQUOI ce module existe à part de la page : la partie calculatoire (types acceptés, calcul de
// redimensionnement) est testable sans monter le composant ni disposer d'un canvas, alors que la
// lecture de fichier ne l'est pas. Les tests portent sur les règles, l'E2E sur le geste.

import type { ChatImage } from './chat-api';

// Les quatre seuls types que l'API Anthropic accepte. La liste est dupliquée côté backend (qui refuse
// tout le reste avec un code stable) : ici c'est du confort, là-bas c'est la garantie.
export const SUPPORTED_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'] as const;

// MAX_EDGE : plus grand côté envoyé au modèle.
// POURQUOI 2576 : c'est la résolution maximale que le modèle exploite. Au-delà on paie des tokens de
// vision pour des pixels qu'il n'utilise pas. En deçà, on rendrait illisible le petit texte d'un
// formulaire — et c'est précisément ce que l'agent doit lire.
export const MAX_EDGE = 2576;

// MAX_BASE64_BYTES : même plafond que le backend (6 Mo de base64). Vérifié ici aussi pour donner un
// retour immédiat plutôt qu'un aller-retour réseau soldé par un 400.
export const MAX_BASE64_BYTES = 6 * 1024 * 1024;

export function isSupportedImage(type: string): boolean {
  return (SUPPORTED_IMAGE_TYPES as readonly string[]).includes(type);
}

// computeTargetSize : dimensions après plafonnement du grand côté, en préservant le ratio.
// Retourne les dimensions d'origine si l'image tient déjà — on évite alors tout ré-encodage, ce qui
// préserve la netteté du texte (une capture 1440p passe intacte).
export function computeTargetSize(
  width: number,
  height: number,
  maxEdge: number = MAX_EDGE,
): { width: number; height: number; resized: boolean } {
  const grandCote = Math.max(width, height);
  if (grandCote <= maxEdge || grandCote === 0) {
    return { width, height, resized: false };
  }
  const facteur = maxEdge / grandCote;
  return {
    width: Math.round(width * facteur),
    height: Math.round(height * facteur),
    resized: true,
  };
}

// stripDataUrlPrefix : l'API veut du base64 NU, sans l'en-tête `data:image/png;base64,`.
export function stripDataUrlPrefix(dataUrl: string): string {
  const virgule = dataUrl.indexOf(',');
  return virgule >= 0 ? dataUrl.slice(virgule + 1) : dataUrl;
}

// extractImageFiles : récupère les images d'un presse-papier ou d'un glisser-déposer.
// POURQUOI un seul point d'entrée pour les deux : DataTransfer est la même structure dans les deux
// cas, et le collage d'une capture (Windows, macOS, Linux) y dépose un fichier image comme un drop.
export function extractImageFiles(data: DataTransfer | null): File[] {
  if (!data) return [];
  const files: File[] = [];
  for (const item of Array.from(data.items ?? [])) {
    if (item.kind !== 'file') continue;
    const file = item.getAsFile();
    if (file && isSupportedImage(file.type)) files.push(file);
  }
  return files;
}

// fileToChatImage : lit un fichier et le prépare pour l'API.
// COMMENT : 1. lecture en data URL, 2. si l'image tient sous MAX_EDGE on la garde TELLE QUELLE (aucun
// ré-encodage, donc aucun flou ajouté sur du texte de formulaire), 3. sinon on la redessine sur un
// canvas aux dimensions plafonnées.
export async function fileToChatImage(file: File): Promise<ChatImage> {
  if (!isSupportedImage(file.type)) {
    throw new Error('image_type_unsupported');
  }

  const dataUrl = await new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error('image_read_failed'));
    reader.readAsDataURL(file);
  });

  const img = await new Promise<HTMLImageElement>((resolve, reject) => {
    const el = new Image();
    el.onload = () => resolve(el);
    el.onerror = () => reject(new Error('image_read_failed'));
    el.src = dataUrl;
  });

  const cible = computeTargetSize(img.naturalWidth, img.naturalHeight);
  if (!cible.resized) {
    return { media_type: file.type, data: stripDataUrlPrefix(dataUrl) };
  }

  const canvas = document.createElement('canvas');
  canvas.width = cible.width;
  canvas.height = cible.height;
  const ctx = canvas.getContext('2d');
  if (!ctx) return { media_type: file.type, data: stripDataUrlPrefix(dataUrl) };
  ctx.drawImage(img, 0, 0, cible.width, cible.height);

  // Le canvas ne sait pas produire de GIF : on retombe sur PNG, sans perte pour une capture.
  const sortie = file.type === 'image/jpeg' ? 'image/jpeg' : 'image/png';
  return { media_type: sortie, data: stripDataUrlPrefix(canvas.toDataURL(sortie, 0.92)) };
}
