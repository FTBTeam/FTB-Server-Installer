package util

import "testing"

func TestParseInstallerName(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want1 int
		want2 int
	}{
		// the table itself
		{"installer_123 should be 123", "installer_123", 123, 0},
		{"installer_123-1234 should be 123", "installer_123-1234", 123, 0},
		{"installer_123_1234 should be 123, 1234", "installer_123_1234", 123, 1234},
		{"installer_123_1234_5678 should be 123, 1234", "installer_123_1234_5678", 123, 1234},
		{"installer-123-1234 should be error", "installer-123-1234", 0, 0},
		{"installer should be error", "installer", 0, 0},
	}
	// The execution loop
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack, version, err := ParseInstallerName(tt.input)
			if err != nil {
				if err.Error() != "invalid installer name" && tt.want1 != 0 && tt.want2 != 0 {
					t.Errorf("got unexpeced error %s", err)
				}
			}
			if pack != tt.want1 {
				t.Errorf("got %d, want %d", pack, tt.want1)
			}
			if version != tt.want2 {
				t.Errorf("got %d, want %d", version, tt.want2)
			}
		})
	}
}
