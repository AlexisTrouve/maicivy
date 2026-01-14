/**
 * Page de demonstration des effets 3D
 * Test des composants Avatar3D, SkillsGraph3D, ParallaxBackground
 */

// Force dynamic rendering to avoid prerender issues with client-only 3D components
export const dynamic = 'force-dynamic';

import Demo3DContent from './Demo3DContent';

export default function Demo3DPage() {
  return <Demo3DContent />;
}
