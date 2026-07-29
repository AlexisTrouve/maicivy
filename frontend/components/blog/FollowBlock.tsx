'use client';

import React from 'react';
import { SubscribeForm } from './SubscribeForm';

interface FollowBlockProps {
  locale?: string;
}

// FollowBlock regroupe les canaux pour suivre le blog : email (form) + Discord, extensible (X plus tard).
//
// POURQUOI config-driven : le lien d'invitation Discord est fourni via NEXT_PUBLIC_DISCORD_INVITE
// (injecté au build). Tant qu'il n'est pas défini, le bouton Discord ne s'affiche pas — le système est
// prêt, il s'active dès qu'on pose l'invite (pas besoin de toucher au code). La notif d'article côté
// Discord vit dans WanMira (webhook de publication), indépendante de ce bouton.
export function FollowBlock({ locale = 'fr' }: FollowBlockProps) {
  const fr = locale === 'fr';
  const discordInvite = process.env.NEXT_PUBLIC_DISCORD_INVITE;

  return (
    <div className="space-y-3">
      <SubscribeForm locale={locale} />

      {discordInvite && (
        <a
          href={discordInvite}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center justify-center gap-2 rounded-xl border border-[#5865F2]/30 bg-[#5865F2]/5 hover:bg-[#5865F2]/12 px-6 py-4 text-[#5865F2] dark:text-[#a8b3ff] font-medium transition-colors"
        >
          {/* Logo Discord */}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M20.317 4.369A19.79 19.79 0 0 0 16.558 3.2a.075.075 0 0 0-.079.037c-.34.6-.717 1.385-.98 2.003a18.27 18.27 0 0 0-5.005 0 12.6 12.6 0 0 0-.995-2.003.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 5.677 4.37a.07.07 0 0 0-.032.027C2.99 8.355 2.26 12.24 2.62 16.075a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.873-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.1 13.1 0 0 1-1.872-.892.077.077 0 0 1-.008-.128c.126-.094.252-.192.372-.291a.074.074 0 0 1 .077-.01c3.927 1.793 8.18 1.793 12.061 0a.074.074 0 0 1 .078.009c.12.099.246.198.373.292a.077.077 0 0 1-.006.127c-.598.349-1.22.645-1.873.891a.076.076 0 0 0-.04.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.84 19.84 0 0 0 6.002-3.03.077.077 0 0 0 .032-.055c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.028ZM8.02 13.331c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.418 2.157-2.418 1.21 0 2.176 1.094 2.157 2.418 0 1.334-.955 2.419-2.157 2.419Zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.418 2.157-2.418 1.21 0 2.176 1.094 2.157 2.418 0 1.334-.946 2.419-2.157 2.419Z" />
          </svg>
          {fr ? 'Rejoindre le Discord' : 'Join the Discord'}
        </a>
      )}
    </div>
  );
}
