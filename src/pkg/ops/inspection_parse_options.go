package ops

import "os"

// DefaultParseInspectionOptions returns commercial defaults for inspection parsing.
// Legacy regex score extraction is disabled unless OPENOCTA_INSPECTION_ALLOW_LEGACY_SCORE=1.
func DefaultParseInspectionOptions() ParseInspectionOptions {
	return ParseInspectionOptions{
		AllowLegacyTextScore: os.Getenv("OPENOCTA_INSPECTION_ALLOW_LEGACY_SCORE") == "1",
	}
}
