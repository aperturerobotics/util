import { StringFilter } from './filter.pb.js'

/**
 * Validates the string filter.
 * @param filter The string filter to validate
 * @returns Error message if invalid, null if valid
 */
export function validateStringFilter(
  filter: StringFilter | null | undefined,
): string | null {
  if (!filter) {
    return null
  }

  if (filter.re) {
    try {
      new RegExp(filter.re)
    } catch (error) {
      return `re: ${error instanceof Error ? error.message : 'invalid regex'}`
    }
  }

  return null
}

/**
 * Checks if the given value matches the string filter.
 * All of the non-zero rules must match for the filter to match.
 * An empty filter matches any.
 * @param filter The string filter to check against
 * @param value The string value to check
 * @returns True if the value matches the filter, false otherwise
 */
export function checkStringFilterMatch(
  filter: StringFilter | null | undefined,
  value: string,
): boolean {
  if (!filter) {
    return true
  }

  if (filter.empty && value !== '') {
    return false
  }

  if (filter.notEmpty && value === '') {
    return false
  }

  if (filter.value && value !== filter.value) {
    return false
  }

  if (
    filter.values &&
    filter.values.length > 0 &&
    !filter.values.includes(value)
  ) {
    return false
  }

  if (filter.re) {
    try {
      const regex = new RegExp(filter.re)
      if (!regex.test(value)) {
        return false
      }
    } catch {
      // Invalid regex treated as a fail (checked in validate but treat as fail)
      return false
    }
  }

  if (filter.hasPrefix && !value.startsWith(filter.hasPrefix)) {
    return false
  }

  if (filter.hasSuffix && !value.endsWith(filter.hasSuffix)) {
    return false
  }

  if (filter.contains && !value.includes(filter.contains)) {
    return false
  }

  return true
}
