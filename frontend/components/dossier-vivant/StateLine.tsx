import type { AgentStatus } from '@/lib/dossier-vivant/events';

import { Icon } from './icons';
import styles from './dossier-vivant.module.css';

/**
 * StateLine — la ligne d'état d'un agent ou du dossier : un glyphe (cercle, arc qui tourne,
 * coche) puis un libellé. L'arc tourne avec l'horloge du run (`now`) — il s'arrête donc avec la
 * pause, comme tout le reste de l'écran.
 */
export function StateLine({
  status,
  label,
  now,
  className = styles.state,
  testId,
}: {
  status: AgentStatus;
  label: string;
  now: number;
  className?: string;
  testId?: string;
}) {
  const icon = status === 'working' ? 'arc' : status === 'done' ? 'check' : 'wait';
  return (
    <p className={className} data-status={status} data-testid={testId}>
      <Icon name={icon} rotate={status === 'working' ? (now * 300) % 360 : undefined} />
      <span>{label}</span>
    </p>
  );
}
