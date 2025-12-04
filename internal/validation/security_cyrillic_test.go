package validation

import (
	"testing"
)

func TestUnicodeSecurity_CyrillicNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		reason   string
	}{
		// Pure Cyrillic names should be ALLOWED
		{
			name:    "pure_cyrillic_first_name",
			input:   "Иван",
			wantErr: false,
			reason:  "Pure Cyrillic name should be allowed",
		},
		{
			name:    "pure_cyrillic_last_name",
			input:   "Петров",
			wantErr: false,
			reason:  "Pure Cyrillic last name should be allowed",
		},
		{
			name:    "pure_cyrillic_full_name",
			input:   "Иван Петров",
			wantErr: false,
			reason:  "Pure Cyrillic full name should be allowed",
		},
		{
			name:    "cyrillic_with_spaces_and_hyphens",
			input:   "Анна-Мария Иванова-Петрова",
			wantErr: false,
			reason:  "Cyrillic with common punctuation should be allowed",
		},

		// Pure Latin names should be ALLOWED
		{
			name:    "pure_latin_first_name",
			input:   "John",
			wantErr: false,
			reason:  "Pure Latin name should be allowed",
		},
		{
			name:    "pure_latin_last_name",
			input:   "Smith",
			wantErr: false,
			reason:  "Pure Latin last name should be allowed",
		},

		// Mixed Cyrillic + Latin homographs should be BLOCKED
		{
			name:    "mixed_homograph_attack_1",
			input:   "аdmin", // Cyrillic а + Latin dmin
			wantErr: true,
			reason:  "Cyrillic 'а' mixed with Latin should be blocked",
		},
		{
			name:    "mixed_homograph_attack_2",
			input:   "РауРаl", // Cyrillic Р + Latin aуРа + Cyrillic l
			wantErr: true,
			reason:  "Cyrillic 'Р' and 'l' mixed with Latin should be blocked",
		},
		{
			name:    "mixed_homograph_attack_3",
			input:   "Gооglе", // Latin G + Cyrillic оо + Latin gl + Cyrillic е
			wantErr: true,
			reason:  "Cyrillic 'о' and 'е' mixed with Latin should be blocked",
		},
		{
			name:    "mixed_homograph_attack_4",
			input:   "Аdmin", // Cyrillic А + Latin dmin
			wantErr: true,
			reason:  "Cyrillic 'А' mixed with Latin should be blocked",
		},

		// Legitimate mixed script (not homographs) should be ALLOWED
		{
			name:    "legitimate_mixed_script",
			input:   "Alex-Алексей", // Latin Alex + hyphen + Cyrillic Алексей
			wantErr: false,
			reason:  "Legitimate mixed script without homographs should be allowed",
		},

		// Edge cases
		{
			name:    "empty_string",
			input:   "",
			wantErr: false,
			reason:  "Empty string should be allowed",
		},
		{
			name:    "only_numbers",
			input:   "12345",
			wantErr: false,
			reason:  "Numbers should be allowed",
		},
		{
			name:    "common_punctuation",
			input:   "john.doe@example.com",
			wantErr: false,
			reason:  "Email-like format should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUnicodeSecurity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUnicodeSecurity() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("Reason: %s", tt.reason)
			}
		})
	}
}

func TestContainsHomographAttacks_SpecificCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		reason   string
	}{
		// Should be ALLOWED (false = no attack) - pure Cyrillic is allowed
		{
			name:     "admin_with_cyrillic_a",
			input:    "админ",
			expected: false,
			reason:   "Pure Cyrillic word should be allowed (no mixed characters)",
		},
		{
			name:     "google_with_cyrillic",
			input:    "gооgle",
			expected: true,
			reason:   "Cyrillic 'о' mixed with Latin",
		},
		{
			name:     "paypal_with_cyrillic",
			input:    "раураl",
			expected: true,
			reason:   "Cyrillic 'р', 'а', 'у' mixed with Latin",
		},
		{
			name:     "mixed_homograph_multiple",
			input:    "Саseу", // Cyrillic С + Latin asey
			expected: true,
			reason:   "Cyrillic 'С' mixed with Latin",
		},

		// Should be ALLOWED (false = no attack)
		{
			name:     "pure_russian_name",
			input:    "Александра",
			expected: false,
			reason:   "Pure Cyrillic name without Latin mixing",
		},
		{
			name:     "pure_russian_text",
			input:    "Привет мир",
			expected: false,
			reason:   "Pure Cyrillic text without Latin mixing",
		},
		{
			name:     "pure_english_name",
			input:    "Alexander",
			expected: false,
			reason:   "Pure Latin name",
		},
		{
			name:     "mixed_legitimate",
			input:    "John-Иван", // Latin John + hyphen + Cyrillic Иван
			expected: false,
			reason:   "Legitimate mixed script without homograph characters",
		},
		{
			name:     "cyrillic_non_homograph",
			input:    "Женя", // Cyrillic ж, е, н, я - none are homographs
			expected: false,
			reason:   "Cyrillic without homograph characters should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHomographAttacks(tt.input)
			if result != tt.expected {
				t.Errorf("containsHomographAttacks() = %v, expected %v", result, tt.expected)
				t.Errorf("Input: %q", tt.input)
				t.Errorf("Reason: %s", tt.reason)
			}
		})
	}
}

func TestValidateFieldSecurity_CyrillicNames(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		fieldName string
		maxLen    int
		wantErr   bool
	}{
		{
			name:      "valid_cyrillic_name",
			field:     "Иван Петров",
			fieldName: "first_name",
			maxLen:    50,
			wantErr:   false,
		},
		{
			name:      "valid_cyrillic_long_name",
			field:     "Александр Александрович",
			fieldName: "full_name",
			maxLen:    50,
			wantErr:   false,
		},
		{
			name:      "suspicious_homograph_mixed",
			field:     "аdmin", // Cyrillic а + Latin dmin
			fieldName: "username",
			maxLen:    50,
			wantErr:   true,
		},
		{
			name:      "field_too_long",
			field:     "ОченьДлинноеИмяКотороеПревышаетДопустимыйЛимитСимволовИПоэтомуДолжноБытьЗаблокировано",
			fieldName: "name",
			maxLen:    50,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFieldSecurity(tt.field, tt.fieldName, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFieldSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}