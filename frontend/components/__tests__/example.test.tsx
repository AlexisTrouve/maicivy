/**
 * Example unit test to verify Jest setup
 * This file can be deleted once you have real component tests
 */

import { render, screen } from '@testing-library/react'

// Simple test component
function HelloWorld() {
  return <h1>Hello, World!</h1>
}

describe('Jest Setup Test', () => {
  it('should render a component', () => {
    render(<HelloWorld />)
    expect(screen.getByText('Hello, World!')).toBeInTheDocument()
  })

  it('should perform basic assertions', () => {
    expect(1 + 1).toBe(2)
    expect('test').toMatch(/test/)
    expect({ name: 'John' }).toHaveProperty('name')
  })

  it('should handle async operations', async () => {
    const promise = Promise.resolve('success')
    await expect(promise).resolves.toBe('success')
  })
})
