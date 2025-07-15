import { describe, it, expect } from 'vitest'
import { validateStringFilter, checkStringFilterMatch } from './filter.js'
import { StringFilter } from './filter.pb.js'

describe('validateStringFilter', () => {
  it('should return null for null/undefined filter', () => {
    expect(validateStringFilter(null)).toBeNull()
    expect(validateStringFilter(undefined)).toBeNull()
  })

  it('should return null for empty filter', () => {
    expect(validateStringFilter({})).toBeNull()
  })

  it('should return null for valid regex', () => {
    expect(validateStringFilter({ re: '^test.*' })).toBeNull()
    expect(validateStringFilter({ re: '[0-9]+' })).toBeNull()
  })

  it('should return error message for invalid regex', () => {
    const result = validateStringFilter({ re: '[' })
    expect(result).toContain('re:')
    expect(result).toContain('Invalid regular expression')
  })

  it('should handle regex with escape sequences', () => {
    expect(validateStringFilter({ re: '\\d+' })).toBeNull()
  })
})

describe('checkStringFilterMatch', () => {
  it('should return true for null/undefined filter', () => {
    expect(checkStringFilterMatch(null, 'test')).toBe(true)
    expect(checkStringFilterMatch(undefined, 'test')).toBe(true)
  })

  it('should return true for empty filter', () => {
    expect(checkStringFilterMatch({}, 'test')).toBe(true)
  })

  describe('empty filter', () => {
    it('should match empty string when empty=true', () => {
      expect(checkStringFilterMatch({ empty: true }, '')).toBe(true)
    })

    it('should not match non-empty string when empty=true', () => {
      expect(checkStringFilterMatch({ empty: true }, 'test')).toBe(false)
    })
  })

  describe('notEmpty filter', () => {
    it('should match non-empty string when notEmpty=true', () => {
      expect(checkStringFilterMatch({ notEmpty: true }, 'test')).toBe(true)
    })

    it('should not match empty string when notEmpty=true', () => {
      expect(checkStringFilterMatch({ notEmpty: true }, '')).toBe(false)
    })
  })

  describe('value filter', () => {
    it('should match exact value', () => {
      expect(checkStringFilterMatch({ value: 'test' }, 'test')).toBe(true)
    })

    it('should not match different value', () => {
      expect(checkStringFilterMatch({ value: 'test' }, 'other')).toBe(false)
    })
  })

  describe('values filter', () => {
    it('should match when value is in values array', () => {
      expect(checkStringFilterMatch({ values: ['test1', 'test2'] }, 'test1')).toBe(true)
      expect(checkStringFilterMatch({ values: ['test1', 'test2'] }, 'test2')).toBe(true)
    })

    it('should not match when value is not in values array', () => {
      expect(checkStringFilterMatch({ values: ['test1', 'test2'] }, 'test3')).toBe(false)
    })

    it('should return true when values array is empty', () => {
      expect(checkStringFilterMatch({ values: [] }, 'test')).toBe(true)
    })
  })

  describe('regex filter', () => {
    it('should match when regex matches', () => {
      expect(checkStringFilterMatch({ re: '^test' }, 'test123')).toBe(true)
      expect(checkStringFilterMatch({ re: '\\d+' }, 'abc123')).toBe(true)
    })

    it('should not match when regex does not match', () => {
      expect(checkStringFilterMatch({ re: '^test' }, 'abc123')).toBe(false)
      expect(checkStringFilterMatch({ re: '\\d+' }, 'abcdef')).toBe(false)
    })

    it('should return false for invalid regex', () => {
      expect(checkStringFilterMatch({ re: '[' }, 'test')).toBe(false)
    })
  })

  describe('hasPrefix filter', () => {
    it('should match when string has prefix', () => {
      expect(checkStringFilterMatch({ hasPrefix: 'test' }, 'test123')).toBe(true)
    })

    it('should not match when string does not have prefix', () => {
      expect(checkStringFilterMatch({ hasPrefix: 'test' }, 'abc123')).toBe(false)
    })
  })

  describe('hasSuffix filter', () => {
    it('should match when string has suffix', () => {
      expect(checkStringFilterMatch({ hasSuffix: '123' }, 'test123')).toBe(true)
    })

    it('should not match when string does not have suffix', () => {
      expect(checkStringFilterMatch({ hasSuffix: '123' }, 'test456')).toBe(false)
    })
  })

  describe('contains filter', () => {
    it('should match when string contains substring', () => {
      expect(checkStringFilterMatch({ contains: 'est' }, 'test123')).toBe(true)
    })

    it('should not match when string does not contain substring', () => {
      expect(checkStringFilterMatch({ contains: 'xyz' }, 'test123')).toBe(false)
    })
  })

  describe('combined filters', () => {
    it('should match when all filters match', () => {
      expect(checkStringFilterMatch({ 
        notEmpty: true, 
        hasPrefix: 'test', 
        hasSuffix: '123' 
      }, 'test123')).toBe(true)
    })

    it('should not match when any filter fails', () => {
      expect(checkStringFilterMatch({ 
        notEmpty: true, 
        hasPrefix: 'test', 
        hasSuffix: '456' 
      }, 'test123')).toBe(false)
    })

    it('should handle complex combination', () => {
      expect(checkStringFilterMatch({ 
        re: '^test', 
        contains: '12', 
        hasSuffix: '3' 
      }, 'test123')).toBe(true)
    })
  })
})
