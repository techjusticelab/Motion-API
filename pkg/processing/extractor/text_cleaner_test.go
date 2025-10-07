package extractor

import (
	"strings"
	"testing"
)

func TestTextCleaner_FilePathArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "File path with timestamp",
			input:    "data/data/data/LAMISD/ARANDA.txtWed Apr 30 19:01:40 2025\nActual document content starts here",
			expected: "Actual document content starts here",
		},
		{
			name:     "Nested data path without timestamp",
			input:    "data/data/data/filename.txt\nDocument content",
			expected: "Document content",
		},
		{
			name:     "Simple file path at start",
			input:    "document.pdf\nThis is the actual content",
			expected: "This is the actual content",
		},
		{
			name:     "Multiple file paths",
			input:    "data/file1.txt\ndata/data/file2.pdf\nActual content line 1\nActual content line 2",
			expected: "Actual content line 1\nActual content line 2",
		},
		{
			name:     "No file paths",
			input:    "This is normal text\nWith multiple lines\nNo artifacts here",
			expected: "This is normal text\nWith multiple lines\nNo artifacts here",
		},
	}

	cleaner := NewTextCleaner(CleaningConfig{
		RemoveFilePathArtifacts: true,
		DebugLogging:           false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.removeFilePathArtifacts(tt.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func TestTextCleaner_HTMLContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic HTML tags",
			input:    "<html><body bgcolor=ffffff><h1><center>Chapter 9<br>Functional Neuroimaging in Psychiatry<br></h1><h2><i>Kathryn J. Kotrla, M.D.</i></h2></center></body></html>",
			expected: "Chapter 9 Functional Neuroimaging in Psychiatry  Kathryn J. Kotrla, M.D.",
		},
		{
			name:     "HTML entities",
			input:    "Text with &nbsp; spaces and &amp; ampersands &lt;brackets&gt;",
			expected: "Text with  spaces and & ampersands <brackets>",
		},
		{
			name:     "Mixed HTML and text",
			input:    "Normal text <p>with paragraph</p> and <strong>bold text</strong>",
			expected: "Normal text with paragraph and bold text",
		},
		{
			name:     "Numeric HTML entities",
			input:    "Text with &#8220;quotes&#8221; and &#8212;dashes&#8212;",
			expected: "Text with quotes and dashes",
		},
		{
			name:     "No HTML content",
			input:    "Plain text with no HTML",
			expected: "Plain text with no HTML",
		},
	}

	cleaner := NewTextCleaner(CleaningConfig{
		RemoveHTMLContent: true,
		DebugLogging:     false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.removeHTMLContent(tt.input)
			result = strings.TrimSpace(strings.ReplaceAll(result, "  ", " "))
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func TestTextCleaner_PrinterArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HP LaserJet artifacts",
			input:    "^4<8|,<<\\0808dddddddddd88,\\t'pll@8@\\,dl'ld0lh(('(hlll<\\4hXX\\X\\\\\\tdtdtdtdtd'tdtdtdtd,(,(,(,(hl\\tdltltdtdtdltdtdtdtdt''('('('('(x<x<x<t\\t\\t\\t\\h4h4h4l\\lXlXlXhx<t\\h4l\\l\\\\\\<'PPddHP LaserJet IIIHPLASIII.PRS\nActual document content",
			expected: "Actual document content",
		},
		{
			name:     "Swiss font artifacts",
			input:    "Swiss Roman 11pt (HP Roman 8) (Port) (FW)\nSwiss Bold 11pt (HP Roman 8) (Port) (FW)\nDocument text here",
			expected: "Document text here",
		},
		{
			name:     "Control sequences",
			input:    "Text with \\4444 control \\sequences and ^4<control> codes",
			expected: "Text with  control  and  codes",
		},
		{
			name:     "No printer artifacts",
			input:    "Clean text without any printer artifacts",
			expected: "Clean text without any printer artifacts",
		},
	}

	cleaner := NewTextCleaner(CleaningConfig{
		RemovePrinterArtifacts: true,
		DebugLogging:          false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.removePrinterArtifacts(tt.input)
			result = strings.TrimSpace(strings.ReplaceAll(result, "  ", " "))
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func TestTextCleaner_SequentialNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Leading sequential numbers",
			input:    "1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 L AW O FFI",
			expected: "L AW O FFI",
		},
		{
			name:     "Sequential number line removal",
			input:    "Document title\n1 2 3 4 5 6 7 8 9 10 11 12 13 14 15\nActual content starts here",
			expected: "Document title\nActual content starts here",
		},
		{
			name:     "Mixed content with numbers",
			input:    "Page 1 of 10\nContent here\n2 3 4 5 6 7 8 More content",
			expected: "Page 1 of 10\nContent here\nMore content",
		},
		{
			name:     "No sequential numbers",
			input:    "Normal text with number 5 and another number 10",
			expected: "Normal text with number 5 and another number 10",
		},
	}

	cleaner := NewTextCleaner(CleaningConfig{
		RemoveSequentialNumbers: true,
		DebugLogging:           false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.removeSequentialNumbers(tt.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func TestTextCleaner_DrivePathReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Windows drive path",
			input:    "C:\\Data\\0Lorman Seminar\nDocument content here",
			expected: "Document content here",
		},
		{
			name:     "Multiple drive paths",
			input:    "File located at D:\\Documents\\Legal\\Case.pdf and backup at E:\\Backup\\Legal\\Case.pdf\nContent follows",
			expected: "File located at  and backup at \nContent follows",
		},
		{
			name:     "UNC path",
			input:    "Network path \\\\server\\share\\documents\\file.doc\nDocument text",
			expected: "Network path \nDocument text",
		},
		{
			name:     "No drive paths",
			input:    "Normal text without any file paths",
			expected: "Normal text without any file paths",
		},
	}

	cleaner := NewTextCleaner(CleaningConfig{
		RemoveDrivePathReferences: true,
		DebugLogging:             false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.removeDrivePathReferences(tt.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func TestTextCleaner_FullCleaningPipeline(t *testing.T) {
	input := `data/data/data/!KOTRLAF.txtWed Apr 30 18:55:26 2025<html><body bgcolor=ffffff><h1><center>Chapter 9<br>Functional Neuroimaging in Psychiatry<br></h1><h2><i>Kathryn J. Kotrla, M.D.</i></h2></center>
C:\Data\Documents\Sample.pdf
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 L AW O FFI
HP LaserJet IIIHPLASIII.PRS ^4<8|,<<\0808dddddddddd88
Swiss Roman 11pt (HP Roman 8) (Port) (FW)

This is the actual document content that should be preserved.
It contains multiple paragraphs and should remain intact.

The cleaning process should remove all artifacts but keep this text.`

	expected := `Chapter 9 Functional Neuroimaging in Psychiatry Kathryn J. Kotrla, M.D.

L AW O FFI

This is the actual document content that should be preserved.
It contains multiple paragraphs and should remain intact.

The cleaning process should remove all artifacts but keep this text.`

	cleaner := NewTextCleaner(DefaultCleaningConfig())
	result := cleaner.CleanText(input)

	// Normalize whitespace for comparison
	result = strings.TrimSpace(strings.ReplaceAll(result, "  ", " "))
	expected = strings.TrimSpace(expected)

	if result != expected {
		t.Errorf("Full cleaning pipeline failed.\nExpected:\n%q\nGot:\n%q", expected, result)
	}
}

func TestTextCleaner_isSequentialNumberLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Clear sequential numbers",
			input:    "1 2 3 4 5 6 7 8 9 10 11 12",
			expected: true,
		},
		{
			name:     "Sequential with text at end",
			input:    "1 2 3 4 5 6 7 LAW OFFICE",
			expected: true,
		},
		{
			name:     "Non-sequential numbers",
			input:    "1 5 10 15 20 25 30",
			expected: false,
		},
		{
			name:     "Too few numbers",
			input:    "1 2 3 4",
			expected: false,
		},
		{
			name:     "Normal text",
			input:    "This is normal text",
			expected: false,
		},
		{
			name:     "Mixed sequential and non-sequential",
			input:    "1 2 3 4 5 100 200 300",
			expected: false,
		},
	}

	cleaner := NewTextCleaner(DefaultCleaningConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.isSequentialNumberLine(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t for input: %q", tt.expected, result, tt.input)
			}
		})
	}
}

func TestTextCleaner_Configuration(t *testing.T) {
	input := "data/data/file.txt<html>Content</html>HP LaserJet"

	// Test with all cleaning disabled
	disabledConfig := CleaningConfig{
		RemoveFilePathArtifacts:   false,
		RemoveHTMLContent:         false,
		RemovePrinterArtifacts:    false,
		RemoveSequentialNumbers:   false,
		RemoveDrivePathReferences: false,
		DebugLogging:             false,
	}

	cleaner := NewTextCleaner(disabledConfig)
	result := cleaner.CleanText(input)

	// Should be mostly unchanged (only final cleanup)
	if !strings.Contains(result, "data/data/file.txt") {
		t.Error("File path should be preserved when RemoveFilePathArtifacts is false")
	}
	if !strings.Contains(result, "<html>") {
		t.Error("HTML should be preserved when RemoveHTMLContent is false")
	}
	if !strings.Contains(result, "HP LaserJet") {
		t.Error("Printer artifacts should be preserved when RemovePrinterArtifacts is false")
	}

	// Test with selective cleaning
	selectiveConfig := CleaningConfig{
		RemoveFilePathArtifacts:   true,
		RemoveHTMLContent:         true,
		RemovePrinterArtifacts:    false,
		RemoveSequentialNumbers:   false,
		RemoveDrivePathReferences: false,
		DebugLogging:             false,
	}

	selectiveCleaner := NewTextCleaner(selectiveConfig)
	selectiveResult := selectiveCleaner.CleanText(input)

	if strings.Contains(selectiveResult, "data/data/file.txt") {
		t.Error("File path should be removed when RemoveFilePathArtifacts is true")
	}
	if strings.Contains(selectiveResult, "<html>") {
		t.Error("HTML should be removed when RemoveHTMLContent is true")
	}
	if !strings.Contains(selectiveResult, "HP LaserJet") {
		t.Error("Printer artifacts should be preserved when RemovePrinterArtifacts is false")
	}
}

func TestTextCleaner_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			input:    "   \n\n\t  \n   ",
			expected: "",
		},
		{
			name:     "Only artifacts",
			input:    "data/data/file.txt<html></html>HP LaserJet",
			expected: "",
		},
		{
			name:     "Single character",
			input:    "A",
			expected: "A",
		},
		{
			name:     "Unicode content",
			input:    "Document with üñìçødé characters",
			expected: "Document with üñìçødé characters",
		},
	}

	cleaner := NewTextCleaner(DefaultCleaningConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.CleanText(tt.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
			}
		})
	}
}

func BenchmarkTextCleaner_FullPipeline(b *testing.B) {
	input := `data/data/data/document.txtMon Jan 15 10:30:00 2024<html><body><h1>Document Title</h1><p>Content here with &nbsp; entities</p></body></html>
C:\Windows\System32\file.exe
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20
HP LaserJet 4000 Series ^4<control>sequences</control>
Swiss Roman 12pt (HP Roman 8) (Portrait) (FastRes)

This is the actual document content that contains important information.
Multiple paragraphs with various formatting and structure.
Legal document text that should be preserved in its entirety.
More content here with numbers like 42 and dates like January 1, 2024.

Final paragraph with conclusion and recommendations.`

	cleaner := NewTextCleaner(DefaultCleaningConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cleaner.CleanText(input)
	}
}