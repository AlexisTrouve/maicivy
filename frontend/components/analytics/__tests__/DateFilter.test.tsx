import { render, screen, fireEvent } from '@testing-library/react';
import DateFilter from '../DateFilter';

describe('DateFilter', () => {
  it('should render date filter component', () => {
    render(<DateFilter />);

    expect(screen.getByText('Période:')).toBeInTheDocument();
  });

  it('should render all preset buttons', () => {
    render(<DateFilter />);

    expect(screen.getByText("Aujourd'hui")).toBeInTheDocument();
    expect(screen.getByText('7 derniers jours')).toBeInTheDocument();
    expect(screen.getByText('30 derniers jours')).toBeInTheDocument();
    expect(screen.getByText('Tout')).toBeInTheDocument();
  });

  it('should have 7 days selected by default', () => {
    render(<DateFilter />);

    const sevenDaysButton = screen.getByText('7 derniers jours');
    expect(sevenDaysButton).toHaveClass('bg-primary', 'text-primary-foreground');
  });

  it('should change selected preset on button click', () => {
    render(<DateFilter />);

    const todayButton = screen.getByText("Aujourd'hui");
    fireEvent.click(todayButton);

    expect(todayButton).toHaveClass('bg-primary', 'text-primary-foreground');
  });

  it('should switch from 7d to 30d preset', () => {
    render(<DateFilter />);

    const thirtyDaysButton = screen.getByText('30 derniers jours');
    fireEvent.click(thirtyDaysButton);

    expect(thirtyDaysButton).toHaveClass('bg-primary', 'text-primary-foreground');

    // Previous selection should no longer have active styles
    const sevenDaysButton = screen.getByText('7 derniers jours');
    expect(sevenDaysButton).not.toHaveClass('text-primary-foreground');
  });

  it('should render Calendar icon', () => {
    const { container } = render(<DateFilter />);

    const calendarIcon = container.querySelector('svg');
    expect(calendarIcon).toBeInTheDocument();
  });

  it('should display date range for today preset', () => {
    render(<DateFilter />);

    const todayButton = screen.getByText("Aujourd'hui");
    fireEvent.click(todayButton);

    // Should display formatted date range
    const dateRange = screen.getByText(/\d{2}\s\w+\s-\s\d{2}\s\w+\s\d{4}/);
    expect(dateRange).toBeInTheDocument();
  });

  it('should display date range for 7 days preset', () => {
    render(<DateFilter />);

    // Already selected by default
    const dateRange = screen.getByText(/\d{2}\s\w+\s-\s\d{2}\s\w+\s\d{4}/);
    expect(dateRange).toBeInTheDocument();
  });

  it('should display date range for 30 days preset', () => {
    render(<DateFilter />);

    const thirtyDaysButton = screen.getByText('30 derniers jours');
    fireEvent.click(thirtyDaysButton);

    const dateRange = screen.getByText(/\d{2}\s\w+\s-\s\d{2}\s\w+\s\d{4}/);
    expect(dateRange).toBeInTheDocument();
  });

  it('should not display date range for "Tout" preset', () => {
    render(<DateFilter />);

    const allButton = screen.getByText('Tout');
    fireEvent.click(allButton);

    // Should not have date range displayed
    const dateRangeElements = screen.queryByText(/\d{2}\s\w+\s-\s\d{2}\s\w+\s\d{4}/);
    expect(dateRangeElements).not.toBeInTheDocument();
  });

  it('should apply hover styles to inactive buttons', () => {
    const { container } = render(<DateFilter />);

    const todayButton = screen.getByText("Aujourd'hui");
    expect(todayButton).toHaveClass('hover:bg-background');
  });

  it('should render buttons in flex container with gap', () => {
    const { container } = render(<DateFilter />);

    const buttonContainer = container.querySelector('.flex.gap-1.bg-muted.p-1.rounded-md');
    expect(buttonContainer).toBeInTheDocument();
  });

  it('should have correct button count', () => {
    const { container } = render(<DateFilter />);

    const buttons = container.querySelectorAll('button');
    expect(buttons.length).toBe(4); // today, 7d, 30d, all
  });

  it('should format dates in French locale', () => {
    render(<DateFilter />);

    const todayButton = screen.getByText("Aujourd'hui");
    fireEvent.click(todayButton);

    // French month abbreviations (janv., févr., mars, etc.)
    const dateText = screen.getByText(/\d{2}\s\w+/);
    expect(dateText).toBeInTheDocument();
  });

  it('should apply text-sm to buttons', () => {
    const { container } = render(<DateFilter />);

    const buttons = container.querySelectorAll('button');
    buttons.forEach((button) => {
      expect(button).toHaveClass('text-sm');
    });
  });

  it('should apply rounded transition-colors to buttons', () => {
    const { container } = render(<DateFilter />);

    const buttons = container.querySelectorAll('button');
    buttons.forEach((button) => {
      expect(button).toHaveClass('rounded', 'transition-colors');
    });
  });

  it('should render date range with ml-2 margin', () => {
    const { container } = render(<DateFilter />);

    const dateRangeContainer = container.querySelector('.text-xs.text-muted-foreground.ml-2');
    expect(dateRangeContainer).toBeInTheDocument();
  });

  it('should handle multiple preset changes', () => {
    render(<DateFilter />);

    // Change to today
    fireEvent.click(screen.getByText("Aujourd'hui"));
    expect(screen.getByText("Aujourd'hui")).toHaveClass('bg-primary');

    // Change to 30 days
    fireEvent.click(screen.getByText('30 derniers jours'));
    expect(screen.getByText('30 derniers jours')).toHaveClass('bg-primary');
    expect(screen.getByText("Aujourd'hui")).not.toHaveClass('bg-primary');

    // Change to all
    fireEvent.click(screen.getByText('Tout'));
    expect(screen.getByText('Tout')).toHaveClass('bg-primary');
    expect(screen.getByText('30 derniers jours')).not.toHaveClass('bg-primary');
  });

  it('should render in flex layout with items-center', () => {
    const { container } = render(<DateFilter />);

    const mainContainer = container.querySelector('.flex.items-center.gap-2');
    expect(mainContainer).toBeInTheDocument();
  });

  it('should have muted foreground text for label', () => {
    const { container } = render(<DateFilter />);

    const label = screen.getByText('Période:');
    expect(label.parentElement).toHaveClass('text-muted-foreground');
  });

  it('should apply px-3 py-1.5 padding to buttons', () => {
    const { container } = render(<DateFilter />);

    const buttons = container.querySelectorAll('button');
    buttons.forEach((button) => {
      expect(button).toHaveClass('px-3', 'py-1.5');
    });
  });

  it('should display year in date range', () => {
    render(<DateFilter />);

    // Default 7 days should show year
    const dateText = screen.getByText(/\d{4}/);
    expect(dateText).toBeInTheDocument();
  });
});
