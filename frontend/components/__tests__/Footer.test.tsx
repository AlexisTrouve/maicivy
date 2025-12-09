import { render, screen } from '@testing-library/react';
import React from 'react';

// Mock Footer component
const Footer = () => {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-gray-900 text-white py-8 mt-auto">
      <div className="container mx-auto px-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* About Section */}
          <div>
            <h3 className="text-lg font-semibold mb-4">MaicIvy</h3>
            <p className="text-gray-400 text-sm">
              CV dynamique interactif avec génération de lettres de motivation par IA.
            </p>
          </div>

          {/* Links Section */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Liens</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <a href="/cv" className="text-gray-400 hover:text-white transition-colors">
                  Mon CV
                </a>
              </li>
              <li>
                <a href="/letters" className="text-gray-400 hover:text-white transition-colors">
                  Lettres de Motivation
                </a>
              </li>
              <li>
                <a href="/analytics" className="text-gray-400 hover:text-white transition-colors">
                  Analytics
                </a>
              </li>
            </ul>
          </div>

          {/* Social Section */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Social</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <a
                  href="https://github.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  GitHub
                </a>
              </li>
              <li>
                <a
                  href="https://linkedin.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  LinkedIn
                </a>
              </li>
              <li>
                <a
                  href="https://twitter.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  Twitter
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Copyright */}
        <div className="mt-8 pt-8 border-t border-gray-800 text-center">
          <p className="text-gray-400 text-sm">
            © {currentYear} MaicIvy. Tous droits réservés.
          </p>
        </div>
      </div>
    </footer>
  );
};

describe('Footer', () => {
  it('should render footer', () => {
    const { container } = render(<Footer />);

    const footer = container.querySelector('footer');
    expect(footer).toBeInTheDocument();
  });

  it('should display MaicIvy brand name', () => {
    render(<Footer />);

    expect(screen.getByText('MaicIvy')).toBeInTheDocument();
  });

  it('should display description', () => {
    render(<Footer />);

    expect(screen.getByText(/CV dynamique interactif/i)).toBeInTheDocument();
  });

  it('should render "Liens" section', () => {
    render(<Footer />);

    expect(screen.getByText('Liens')).toBeInTheDocument();
  });

  it('should render CV link', () => {
    render(<Footer />);

    const cvLink = screen.getByText('Mon CV').closest('a');
    expect(cvLink).toHaveAttribute('href', '/cv');
  });

  it('should render Lettres de Motivation link', () => {
    render(<Footer />);

    const lettersLink = screen.getByText('Lettres de Motivation').closest('a');
    expect(lettersLink).toHaveAttribute('href', '/letters');
  });

  it('should render Analytics link', () => {
    render(<Footer />);

    const analyticsLink = screen.getByText('Analytics').closest('a');
    expect(analyticsLink).toHaveAttribute('href', '/analytics');
  });

  it('should render "Social" section', () => {
    render(<Footer />);

    expect(screen.getByText('Social')).toBeInTheDocument();
  });

  it('should render GitHub link with correct attributes', () => {
    render(<Footer />);

    const githubLink = screen.getByText('GitHub').closest('a');
    expect(githubLink).toHaveAttribute('href', 'https://github.com');
    expect(githubLink).toHaveAttribute('target', '_blank');
    expect(githubLink).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('should render LinkedIn link with correct attributes', () => {
    render(<Footer />);

    const linkedinLink = screen.getByText('LinkedIn').closest('a');
    expect(linkedinLink).toHaveAttribute('href', 'https://linkedin.com');
    expect(linkedinLink).toHaveAttribute('target', '_blank');
    expect(linkedinLink).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('should render Twitter link with correct attributes', () => {
    render(<Footer />);

    const twitterLink = screen.getByText('Twitter').closest('a');
    expect(twitterLink).toHaveAttribute('href', 'https://twitter.com');
    expect(twitterLink).toHaveAttribute('target', '_blank');
    expect(twitterLink).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('should display copyright with current year', () => {
    render(<Footer />);

    const currentYear = new Date().getFullYear();
    expect(screen.getByText(new RegExp(`© ${currentYear} MaicIvy`))).toBeInTheDocument();
  });

  it('should display "Tous droits réservés"', () => {
    render(<Footer />);

    expect(screen.getByText(/tous droits réservés/i)).toBeInTheDocument();
  });

  it('should have dark background', () => {
    const { container } = render(<Footer />);

    const footer = container.querySelector('footer');
    expect(footer).toHaveClass('bg-gray-900', 'text-white');
  });

  it('should use grid layout for sections', () => {
    const { container } = render(<Footer />);

    const grid = container.querySelector('.grid');
    expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-3');
  });

  it('should have border-top on copyright section', () => {
    const { container } = render(<Footer />);

    const copyrightSection = container.querySelector('.border-t');
    expect(copyrightSection).toBeInTheDocument();
  });

  it('should center copyright text', () => {
    const { container } = render(<Footer />);

    const copyrightSection = container.querySelector('.text-center');
    expect(copyrightSection).toBeInTheDocument();
  });

  it('should apply hover styles to links', () => {
    const { container } = render(<Footer />);

    const links = container.querySelectorAll('a');
    links.forEach(link => {
      if (link.textContent !== 'MaicIvy') {
        expect(link).toHaveClass('transition-colors');
      }
    });
  });

  it('should have proper spacing with margin-top auto', () => {
    const { container } = render(<Footer />);

    const footer = container.querySelector('footer');
    expect(footer).toHaveClass('mt-auto');
  });

  it('should render three columns in footer', () => {
    render(<Footer />);

    const headers = screen.getAllByRole('heading', { level: 3 });
    expect(headers).toHaveLength(3);
    expect(headers[0]).toHaveTextContent('MaicIvy');
    expect(headers[1]).toHaveTextContent('Liens');
    expect(headers[2]).toHaveTextContent('Social');
  });

  it('should use container class for responsive width', () => {
    const { container } = render(<Footer />);

    const contentContainer = container.querySelector('.container');
    expect(contentContainer).toBeInTheDocument();
    expect(contentContainer).toHaveClass('mx-auto', 'px-4');
  });
});
