package fs

import "testing"

func TestPrepareFileExt(t *testing.T) {
	var tests = []struct {
		name       string
		ext        string
		targetName string
		targetExt  string
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

	for _, test := range tests {
		_name, _ext := prepareFileExt(test.name, test.ext)
		if _name != test.targetName || _ext != test.targetExt {
			t.Errorf("invalid file name prepare [%s != %s] or [%s != %s]", _name, test.targetName, _ext, test.targetExt)
		}
	}
}
