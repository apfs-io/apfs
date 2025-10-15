package fs

import "testing"

// TestPrepareFileExt tests the prepareFileExt function with various file name and extension combinations.
func TestPrepareFileExt(t *testing.T) {
	// Define test cases
	var tests = []struct {
		name       string // Input file name
		ext        string // Extension to apply
		targetName string // Expected base name after processing
		targetExt  string // Expected extension after processing
	}{
		{
			name:       "icon.png",
			ext:        ".png",
			targetName: "icon",
			targetExt:  ".png",
		},
		{
			name:       "base/banner.jpeg",
			ext:        ".jpg",
			targetName: "base/banner",
			targetExt:  ".jpg",
		},
		{
			name:       "super-puper.application",
			ext:        ".app",
			targetName: "super-puper",
			targetExt:  ".app",
		},
		{
			name:       "Camel-Case.File.Extantion",
			ext:        ".EXT",
			targetName: "Camel-Case.File",
			targetExt:  ".ext",
		},
		{
			name:       "try",
			ext:        "bin",
			targetName: "try",
			targetExt:  ".bin",
		},
	}

	// Run each test case
	for _, test := range tests {
		_name, _ext := prepareFileExt(test.name, test.ext)
		if _name != test.targetName || _ext != test.targetExt {
			t.Errorf(
				"invalid file name prepare [%s != %s] or [%s != %s]",
				_name, test.targetName, _ext, test.targetExt,
			)
		}
	}
}
