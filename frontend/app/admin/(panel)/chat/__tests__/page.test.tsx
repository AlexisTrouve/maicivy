import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AdminChatTool from '../page';

// streamChat mocké : simule une réponse de l'agent (un tool_call + du texte + done).
jest.mock('@/lib/chat-api', () => ({
  streamChat: jest.fn(async (message: string, history: unknown, cb: any) => {
    cb.onToolCall('search_projects', {});
    cb.onText('Voici ce que je trouve.');
    cb.onDone();
  }),
}));

import { streamChat } from '@/lib/chat-api';

function mockFetch() {
  global.fetch = jest.fn((url: RequestInfo | URL, opts?: RequestInit) => {
    const u = String(url);
    const method = opts?.method || 'GET';
    if (u.endsWith('/admin/chat/conversations') && method === 'GET') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ conversations: [] }) } as Response);
    }
    if (u.endsWith('/admin/chat/conversations') && method === 'POST') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 'conv-1', title: '' }) } as Response);
    }
    // PUT save / GET detail / DELETE
    return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 'conv-1', messages: [] }) } as Response);
  }) as jest.Mock;
}

describe('AdminChatTool', () => {
  beforeEach(() => mockFetch());
  afterEach(() => jest.restoreAllMocks());

  it('rend la sidebar conversations + le thread + l’input', async () => {
    render(<AdminChatTool />);
    await waitFor(() => expect(screen.getByTestId('admin-chat-tool')).toBeInTheDocument());
    expect(screen.getByTestId('chat-new')).toBeInTheDocument();
    expect(screen.getByTestId('chat-conversations')).toBeInTheDocument();
    expect(screen.getByTestId('chat-input')).toBeInTheDocument();
  });

  it('envoyer un message → crée une conv, streame, affiche la réponse de l’agent', async () => {
    render(<AdminChatTool />);
    await waitFor(() => expect(screen.getByTestId('chat-input')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('chat-input'), { target: { value: 'Parle-moi de mes projets Go' } });
    fireEvent.click(screen.getByTestId('chat-send'));

    // streamChat appelé avec le message
    await waitFor(() => expect(streamChat).toHaveBeenCalled());
    expect((streamChat as jest.Mock).mock.calls[0][0]).toBe('Parle-moi de mes projets Go');

    // message user + réponse agent affichés
    await waitFor(() => {
      expect(screen.getByTestId('chat-thread')).toHaveTextContent('Parle-moi de mes projets Go');
      expect(screen.getByTestId('chat-thread')).toHaveTextContent('Voici ce que je trouve.');
    });

    // une conversation a été créée (POST) puis sauvegardée (PUT)
    const calls = (global.fetch as jest.Mock).mock.calls.map((c) => `${c[1]?.method || 'GET'} ${c[0]}`);
    expect(calls.some((c) => c.startsWith('POST') && c.includes('/admin/chat/conversations'))).toBe(true);
    expect(calls.some((c) => c.startsWith('PUT') && c.includes('/admin/chat/conversations/'))).toBe(true);
  });
});
