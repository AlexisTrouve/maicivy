import { renderHook, act } from '@testing-library/react'
import { useTheme, ThemeProvider } from '@/components/providers/ThemeProvider'
import React from 'react'

describe('useTheme', () => {
  // Mock localStorage
  let localStorageMock: { [key: string]: string } = {}
  let getItemSpy: jest.Mock
  let setItemSpy: jest.Mock

  // Wrapper helper
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(ThemeProvider, null, children)

  beforeEach(() => {
    localStorageMock = {}

    getItemSpy = jest.fn((key: string) => localStorageMock[key] || null)
    setItemSpy = jest.fn((key: string, value: string) => {
      localStorageMock[key] = value
    })

    Object.defineProperty(global, 'localStorage', {
      value: {
        getItem: getItemSpy,
        setItem: setItemSpy,
        removeItem: jest.fn((key: string) => {
          delete localStorageMock[key]
        }),
        clear: jest.fn(() => {
          localStorageMock = {}
        }),
        key: jest.fn(),
        length: 0,
      },
      writable: true,
    })

    // Mock window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: jest.fn().mockImplementation((query) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
      })),
    })

    // Reset document classes
    document.documentElement.className = ''
  })

  it('should initialize with dark theme by default (no stored pref)', () => {
    // Dark est le défaut produit (esthétique dark-first). Sans choix stocké, le 1er rendu est dark
    // même si la préférence système est claire (matchMedia mocké matches:false → système non-dark).
    const { result } = renderHook(() => useTheme(), { wrapper })

    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('should initialize with stored theme from localStorage', () => {
    localStorageMock['theme'] = 'dark'

    const { result } = renderHook(() => useTheme(), { wrapper })

    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('should respect a stored "light" choice over the dark default', () => {
    // Le choix explicite de l'utilisateur (localStorage, posé par le toggle) gagne TOUJOURS sur le
    // défaut dark — même si la préférence système est sombre (adversarial : on prouve que ni le défaut
    // ni le système ne réécrasent un choix stocké).
    localStorageMock['theme'] = 'light'
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: jest.fn().mockImplementation((query) => ({
        matches: query === '(prefers-color-scheme: dark)',
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
      })),
    })

    const { result } = renderHook(() => useTheme(), { wrapper })

    expect(result.current.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('should toggle theme from light to dark', () => {
    // On part d'un choix stocké "light" (le défaut étant dark désormais) pour exercer la transition
    // light → dark du toggle.
    localStorageMock['theme'] = 'light'

    const { result } = renderHook(() => useTheme(), { wrapper })

    expect(result.current.theme).toBe('light')

    act(() => {
      result.current.toggleTheme()
    })

    expect(result.current.theme).toBe('dark')
    expect(setItemSpy).toHaveBeenCalledWith('theme', 'dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('should toggle theme from dark to light', () => {
    localStorageMock['theme'] = 'dark'

    const { result } = renderHook(() => useTheme(), { wrapper })

    expect(result.current.theme).toBe('dark')

    act(() => {
      result.current.toggleTheme()
    })

    expect(result.current.theme).toBe('light')
    expect(setItemSpy).toHaveBeenCalledWith('theme', 'light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('should persist theme changes to localStorage', () => {
    // Part d'un choix stocké "light" pour que la 1re bascule donne "dark" puis "light" (le défaut
    // étant dark, sans ce seed la 1re bascule donnerait "light" et inverserait les assertions).
    localStorageMock['theme'] = 'light'

    const { result } = renderHook(() => useTheme(), { wrapper })

    act(() => {
      result.current.toggleTheme()
    })

    expect(setItemSpy).toHaveBeenCalledWith('theme', 'dark')

    act(() => {
      result.current.toggleTheme()
    })

    expect(setItemSpy).toHaveBeenCalledWith('theme', 'light')
  })
})
