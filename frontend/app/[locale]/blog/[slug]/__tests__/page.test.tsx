// Page article du blog : seul un 404 du backend mène à « introuvable ».
//
// POURQUOI : toute erreur devenait notFound() — une panne de maiProFiles faisait passer des
// articles existants pour supprimés (Google désindexe un 404, pas une erreur temporaire).

const notFound = jest.fn(() => {
  throw new Error('NEXT_NOT_FOUND');
});
jest.mock('next/navigation', () => ({ notFound: () => notFound() }));

const getPost = jest.fn();
jest.mock('@/lib/api', () => ({
  blogApi: { getPost: (...a: unknown[]) => getPost(...a) },
}));

jest.mock('@/components/blog', () => ({
  BlogPostView: () => null,
}));

import BlogPostPage from '../page';

// Next 15 : les pages reçoivent params/searchParams en Promise — les tests les passent pareil.
const params = Promise.resolve({ locale: 'fr', slug: 'premier' });

describe('BlogPostPage — panne amont vs article absent', () => {
  beforeEach(() => {
    notFound.mockClear();
    getPost.mockReset();
  });

  it('article inexistant (404 backend) → notFound()', async () => {
    getPost.mockRejectedValue({ success: false, statusCode: 404 });

    await expect(BlogPostPage({ params })).rejects.toThrow('NEXT_NOT_FOUND');
    expect(notFound).toHaveBeenCalledTimes(1);
  });

  it('maiProFiles en panne (502 backend) → erreur relancée, PAS de notFound()', async () => {
    const panne = { success: false, statusCode: 502, message: 'Erreur 502' };
    getPost.mockRejectedValue(panne);

    await expect(BlogPostPage({ params })).rejects.toBe(panne);
    expect(notFound).not.toHaveBeenCalled();
  });

  it('erreur réseau sans statut (timeout) → relancée, PAS de notFound()', async () => {
    getPost.mockRejectedValue(new Error('aborted'));

    await expect(BlogPostPage({ params })).rejects.toThrow('aborted');
    expect(notFound).not.toHaveBeenCalled();
  });
});
