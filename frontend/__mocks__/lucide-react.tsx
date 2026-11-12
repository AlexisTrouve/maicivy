// Mock jest GLOBAL de lucide-react via Proxy : TOUTE icône (CheckCircle2, ArrowRight, Layers, Clock…)
// renvoie un <svg data-testid="...-icon">. Évite la liste figée d'avant qui cassait dès qu'un composant
// utilisait une icône non listée ("Element type is invalid: undefined").
// Fichier en CommonJS pur (require/module.exports) pour exporter un Proxy au niveau module.
const React = require('react');

const handler = {
  get(_target, name) {
    if (name === '__esModule') return true;
    if (name === 'default' || typeof name === 'symbol') return undefined;
    // CheckCircle2 -> "check-circle2-icon", ExternalLink -> "external-link-icon", Github -> "github-icon"
    const testid = String(name).replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase() + '-icon';
    return function MockIcon(props) {
      props = props || {};
      // Loader2 garde animate-spin (des tests vérifient le spinner). On fusionne avec la className passée.
      const base = name === 'Loader2' ? 'animate-spin' : '';
      const className = [base, props.className].filter(Boolean).join(' ');
      return React.createElement(
        'svg',
        Object.assign({}, props, { 'data-testid': testid, className: className || undefined })
      );
    };
  },
};

module.exports = new Proxy({}, handler);
