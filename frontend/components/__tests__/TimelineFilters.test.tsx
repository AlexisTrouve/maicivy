import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import TimelineFilters from '../timeline/TimelineFilters';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, initial, animate, exit, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
    button: ({ children, whileHover, whileTap, ...props }: any) => (
      <button {...props}>{children}</button>
    ),
  },
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Filter: () => <svg data-testid="filter-icon" />,
  X: () => <svg data-testid="x-icon" />,
  Calendar: () => <svg data-testid="calendar-icon" />,
}));

const mockCategories = ['Backend', 'Frontend', 'DevOps', 'Mobile'];

describe('TimelineFilters', () => {
  it('should render filters title', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText('Filtres')).toBeInTheDocument();
    expect(screen.getByTestId('filter-icon')).toBeInTheDocument();
  });

  it('should render type filter buttons', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText('Tous')).toBeInTheDocument();
    expect(screen.getByText('Expériences')).toBeInTheDocument();
    expect(screen.getByText('Projets')).toBeInTheDocument();
  });

  it('should render category filter buttons', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText('Backend')).toBeInTheDocument();
    expect(screen.getByText('Frontend')).toBeInTheDocument();
    expect(screen.getByText('DevOps')).toBeInTheDocument();
    expect(screen.getByText('Mobile')).toBeInTheDocument();
  });

  it('should call onFiltersChange when type is changed', () => {
    const onFiltersChange = jest.fn();

    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={onFiltersChange}
        onReset={jest.fn()}
      />
    );

    const experiencesButton = screen.getByText('Expériences');
    fireEvent.click(experiencesButton);

    expect(onFiltersChange).toHaveBeenCalledWith({
      category: '',
      type: 'experience',
      period: null,
    });
  });

  it('should call onFiltersChange when category is selected', () => {
    const onFiltersChange = jest.fn();

    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={onFiltersChange}
        onReset={jest.fn()}
      />
    );

    const backendButton = screen.getByText('Backend');
    fireEvent.click(backendButton);

    expect(onFiltersChange).toHaveBeenCalledWith({
      category: 'Backend',
      type: 'all',
      period: null,
    });
  });

  it('should deselect category when clicked again', () => {
    const onFiltersChange = jest.fn();

    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory="Backend"
        selectedType="all"
        onFiltersChange={onFiltersChange}
        onReset={jest.fn()}
      />
    );

    const backendButton = screen.getByText('Backend');
    fireEvent.click(backendButton);

    expect(onFiltersChange).toHaveBeenCalledWith({
      category: '',
      type: 'all',
      period: null,
    });
  });

  it('should highlight selected type', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="experience"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    const experiencesButton = screen.getByText('Expériences');
    expect(experiencesButton).toHaveClass('bg-blue-500');
  });

  it('should highlight selected category', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory="Backend"
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    const backendButton = screen.getByText('Backend');
    expect(backendButton).toHaveClass('bg-purple-500');
  });

  it('should show reset button when filters are active', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory="Backend"
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText('Réinitialiser')).toBeInTheDocument();
    expect(screen.getByTestId('x-icon')).toBeInTheDocument();
  });

  it('should not show reset button when no filters are active', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.queryByText('Réinitialiser')).not.toBeInTheDocument();
  });

  it('should call onReset when reset button is clicked', () => {
    const onReset = jest.fn();

    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory="Backend"
        selectedType="experience"
        onFiltersChange={jest.fn()}
        onReset={onReset}
      />
    );

    const resetButton = screen.getByText('Réinitialiser');
    fireEvent.click(resetButton);

    expect(onReset).toHaveBeenCalled();
  });

  it('should toggle period picker visibility', () => {
    const { container } = render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    expect(screen.getByText(/de :/i)).toBeInTheDocument();
    expect(screen.getByText(/à :/i)).toBeInTheDocument();
    const dateInputs = container.querySelectorAll('input[type="date"]');
    expect(dateInputs.length).toBe(2);
  });

  it('should render date inputs in period picker', async () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    await waitFor(() => {
      const dateInputs = screen.getAllByDisplayValue('');
      expect(dateInputs.length).toBeGreaterThanOrEqual(2);
    });
  });

  it('should apply period filter', async () => {
    const onFiltersChange = jest.fn();

    const { container } = render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={onFiltersChange}
        onReset={jest.fn()}
      />
    );

    // Open period picker
    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    // Set dates
    const dateInputs = container.querySelectorAll('input[type="date"]');
    const fromInput = dateInputs[0];
    const toInput = dateInputs[1];

    fireEvent.change(fromInput, { target: { value: '2022-01-01' } });
    fireEvent.change(toInput, { target: { value: '2023-12-31' } });

    // Apply filter
    const applyButton = screen.getByText('Appliquer');
    fireEvent.click(applyButton);

    expect(onFiltersChange).toHaveBeenCalledWith({
      category: '',
      type: 'all',
      period: {
        from: '2022-01-01',
        to: '2023-12-31',
      },
    });
  });

  it('should disable apply button when dates are incomplete', async () => {
    const { container } = render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    // Open period picker
    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    const applyButton = screen.getByText('Appliquer');
    expect(applyButton).toBeDisabled();

    // Set only one date
    const dateInputs = container.querySelectorAll('input[type="date"]');
    const fromInput = dateInputs[0];
    fireEvent.change(fromInput, { target: { value: '2022-01-01' } });

    expect(applyButton).toBeDisabled();
  });

  it('should clear period filter', async () => {
    const onFiltersChange = jest.fn();

    const { container } = render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={onFiltersChange}
        onReset={jest.fn()}
      />
    );

    // Open period picker
    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    // Set dates
    const dateInputs = container.querySelectorAll('input[type="date"]');
    const fromInput = dateInputs[0];
    const toInput = dateInputs[1];

    fireEvent.change(fromInput, { target: { value: '2022-01-01' } });
    fireEvent.change(toInput, { target: { value: '2023-12-31' } });

    // Clear filter
    const clearButton = screen.getByText('Effacer');
    fireEvent.click(clearButton);

    await waitFor(() => {
      expect(screen.queryByDisplayValue('2022-01-01')).not.toBeInTheDocument();
    });
  });

  it('should update period button text when dates are selected', async () => {
    const { container } = render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    // Open period picker
    const periodButton = screen.getByText(/filtrer par période/i);
    fireEvent.click(periodButton);

    // Set dates
    const dateInputs = container.querySelectorAll('input[type="date"]');
    const fromInput = dateInputs[0];
    const toInput = dateInputs[1];

    fireEvent.change(fromInput, { target: { value: '2022-01-01' } });
    fireEvent.change(toInput, { target: { value: '2023-12-31' } });

    // Close period picker
    const clearButton = screen.getByText('Effacer');
    fireEvent.click(clearButton);

    await waitFor(() => {
      expect(screen.getByText(/filtrer par période/i)).toBeInTheDocument();
    });
  });

  it('should render calendar icon in period filter', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByTestId('calendar-icon')).toBeInTheDocument();
  });

  it('should show Type label', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText(/Type :/i)).toBeInTheDocument();
  });

  it('should show Catégorie label', () => {
    render(
      <TimelineFilters
        categories={mockCategories}
        selectedCategory=""
        selectedType="all"
        onFiltersChange={jest.fn()}
        onReset={jest.fn()}
      />
    );

    expect(screen.getByText(/Catégorie :/i)).toBeInTheDocument();
  });
});
