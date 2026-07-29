'use client';

import { useState } from 'react';
import { FileText, MessageSquare } from 'lucide-react';

interface LettersTabsProps {
  letterGenerator: React.ReactNode;
  messageGenerator: React.ReactNode;
}

const TABS = [
  { id: 'letter', label: 'Lettre de motivation', icon: FileText },
  { id: 'message', label: 'Message plateforme', icon: MessageSquare },
] as const;

export function LettersTabs({ letterGenerator, messageGenerator }: LettersTabsProps) {
  const [activeTab, setActiveTab] = useState<'letter' | 'message'>('letter');

  return (
    <div>
      {/* Tab selector */}
      <div className="flex justify-center mb-8">
        <div className="inline-flex bg-white dark:bg-slate-800 rounded-xl p-1 shadow border border-slate-200 dark:border-slate-700 gap-1">
          {TABS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setActiveTab(id)}
              className={`flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium transition-all ${
                activeTab === id
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
            >
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      {activeTab === 'letter' ? letterGenerator : messageGenerator}
    </div>
  );
}
