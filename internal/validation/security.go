package validation

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Security validation errors
var (
	ErrInvalidUnicodeSecurity  = fmt.Errorf("ErrInvalidUnicodeSecurity")
	ErrHomographAttackDetected = fmt.Errorf("ErrHomographAttackDetected")
	ErrInvalidUnicodeCategory  = fmt.Errorf("ErrInvalidUnicodeCategory")
)

// Blocked Unicode categories for security
var blockedCategories = []*unicode.RangeTable{
	unicode.Cc, // Control characters
	unicode.Cf, // Format characters (zero-width, etc.)
	unicode.Cs, // Surrogate characters
	unicode.Co, // Private use characters
}

// Dangerous Unicode characters that should be blocked
var dangerousRunes = []rune{
	0x0000, // NULL
	0x0008, // BACKSPACE
	0x000B, // VERTICAL TAB
	0x000C, // FORM FEED
	0x000E, // SHIFT OUT
	0x000F, // SHIFT IN
	0x0010, // DATA LINK ESCAPE
	0x0011, // DEVICE CONTROL ONE
	0x0012, // DEVICE CONTROL TWO
	0x0013, // DEVICE CONTROL THREE
	0x0014, // DEVICE CONTROL FOUR
	0x0015, // NEGATIVE ACKNOWLEDGE
	0x0016, // SYNCHRONOUS IDLE
	0x0017, // END OF TRANSMISSION BLOCK
	0x0018, // CANCEL
	0x0019, // END OF MEDIUM
	0x001A, // SUBSTITUTE
	0x001B, // ESCAPE
	0x001C, // FILE SEPARATOR
	0x001D, // GROUP SEPARATOR
	0x001E, // RECORD SEPARATOR
	0x001F, // UNIT SEPARATOR
	0x007F, // DELETE
}

// ValidateUnicodeSecurity performs comprehensive Unicode security validation
func ValidateUnicodeSecurity(input string) error {
	// Early rejection for obvious dangerous characters
	for _, dangerous := range dangerousRunes {
		if strings.ContainsRune(input, dangerous) {
			return ErrInvalidUnicodeSecurity
		}
	}

	// Unicode normalization with security check
	normalized := norm.NFKC.String(input)

	// Check for homograph attacks in both original and normalized
	if containsHomographAttacks(input) || containsHomographAttacks(normalized) {
		return ErrHomographAttackDetected
	}

	// Check for blocked Unicode categories
	for _, r := range normalized {
		if unicode.IsOneOf(blockedCategories, r) {
			return ErrInvalidUnicodeCategory
		}
	}

	return nil
}

// containsHomographAttacks checks for common homograph attack patterns
// Only blocks mixed strings that use Cyrillic to impersonate Latin, not pure Cyrillic text
func containsHomographAttacks(input string) bool {
	// Mapping of Cyrillic homographs to their Latin lookalikes
	cyrillicToLatin := map[rune]rune{
		'\u0430': 'a', // Cyrillic small а → Latin a
		'\u0435': 'e', // Cyrillic small е → Latin e
		'\u043e': 'o', // Cyrillic small о → Latin o
		'\u0440': 'p', // Cyrillic small р → Latin p
		'\u0441': 'c', // Cyrillic small с → Latin c
		'\u0445': 'x', // Cyrillic small х → Latin x
		'\u0443': 'y', // Cyrillic small у → Latin y
		'\u0410': 'A', // Cyrillic capital А → Latin A
		'\u0415': 'E', // Cyrillic capital Е → Latin E
		'\u041E': 'O', // Cyrillic capital О → Latin O
		'\u0420': 'P', // Cyrillic capital Р → Latin P
		'\u0421': 'C', // Cyrillic capital С → Latin C
		'\u0425': 'X', // Cyrillic capital Х → Latin X
		'\u0423': 'Y', // Cyrillic capital У → Latin Y
	}

	runes := []rune(input)

	// Check for suspicious patterns: Cyrillic homograph directly adjacent to Latin character
	for i, r := range runes {
		// Check if character is a Cyrillic homograph
		if _, isHomograph := cyrillicToLatin[r]; isHomograph {
			// Look at immediate neighbors for Latin letters (not separated by hyphens/spaces)
			if i > 0 {
				prevRune := runes[i-1]
				// Only consider it suspicious if previous character is Latin AND no separator
				if ((prevRune >= 'a' && prevRune <= 'z') || (prevRune >= 'A' && prevRune <= 'Z')) &&
					prevRune != '-' && prevRune != ' ' && prevRune != '.' && prevRune != '_' {
					return true
				}
			}
			if i < len(runes)-1 {
				nextRune := runes[i+1]
				// Only consider it suspicious if next character is Latin AND no separator
				if ((nextRune >= 'a' && nextRune <= 'z') || (nextRune >= 'A' && nextRune <= 'Z')) &&
					nextRune != '-' && nextRune != ' ' && nextRune != '.' && nextRune != '_' {
					return true
				}
			}
		}
	}

	// Allow pure Cyrillic names like "Иван", "Мария", "Александр"
	// Allow pure Latin names like "John", "Maria", "Alexander"
	// Allow legitimate mixed like "Alex-Алексей" where separated by hyphens
	return false
}

// ValidateFieldSecurity validates a single field with length and security checks
func ValidateFieldSecurity(field, fieldName string, maxLen int) error {
	// Check length limits
	if len(field) > maxLen {
		return fmt.Errorf("field %s exceeds maximum length of %d characters", fieldName, maxLen)
	}

	// Unicode security validation
	if err := ValidateUnicodeSecurity(field); err != nil {
		return fmt.Errorf("unicode security validation failed for field %s: %w", fieldName, err)
	}

	return nil
}
