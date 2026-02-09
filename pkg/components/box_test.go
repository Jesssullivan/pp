package components

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Align / Padding tests
// ---------------------------------------------------------------------------

func TestNewPadding(t *testing.T) {
	p := NewPadding(3)
	if p.Top != 3 || p.Right != 3 || p.Bottom != 3 || p.Left != 3 {
		t.Errorf("NewPadding(3) = %+v, want all 3", p)
	}
}

func TestNewPaddingNegativeClamped(t *testing.T) {
	p := NewPadding(-5)
	if p.Top != 0 || p.Right != 0 || p.Bottom != 0 || p.Left != 0 {
		t.Errorf("NewPadding(-5) = %+v, want all 0", p)
	}
}

func TestNewPaddingHV(t *testing.T) {
	p := NewPaddingHV(2, 1)
	if p.Top != 1 || p.Bottom != 1 || p.Left != 2 || p.Right != 2 {
		t.Errorf("NewPaddingHV(2,1) = %+v, want T=1,B=1,L=2,R=2", p)
	}
}

func TestNewPaddingHVNegative(t *testing.T) {
	p := NewPaddingHV(-1, -2)
	if p.Top != 0 || p.Bottom != 0 || p.Left != 0 || p.Right != 0 {
		t.Errorf("NewPaddingHV(-1,-2) = %+v, want all 0", p)
	}
}

// ---------------------------------------------------------------------------
// Style tests
// ---------------------------------------------------------------------------

func TestColorHex(t *testing.T) {
	c := Color("#ff5500")
	want := "\x1b[38;2;255;85;0m"
	if c != want {
		t.Errorf("Color(#ff5500) = %q, want %q", c, want)
	}
}

func TestColorHexNoHash(t *testing.T) {
	c := Color("00ff00")
	want := "\x1b[38;2;0;255;0m"
	if c != want {
		t.Errorf("Color(00ff00) = %q, want %q", c, want)
	}
}

func TestBgColorHex(t *testing.T) {
	c := BgColor("#001122")
	want := "\x1b[48;2;0;17;34m"
	if c != want {
		t.Errorf("BgColor(#001122) = %q, want %q", c, want)
	}
}

func TestColorInvalid(t *testing.T) {
	if c := Color("xyz"); c != "" {
		t.Errorf("Color(xyz) = %q, want empty", c)
	}
	if c := Color(""); c != "" {
		t.Errorf("Color('') = %q, want empty", c)
	}
	if c := Color("#gg0000"); c != "" {
		t.Errorf("Color(#gg0000) = %q, want empty", c)
	}
}

func TestBold(t *testing.T) {
	s := Bold("hi")
	if s != "\x1b[1mhi\x1b[22m" {
		t.Errorf("Bold(hi) = %q", s)
	}
}

func TestDim(t *testing.T) {
	s := Dim("hi")
	if s != "\x1b[2mhi\x1b[22m" {
		t.Errorf("Dim(hi) = %q", s)
	}
}

func TestItalic(t *testing.T) {
	s := Italic("hi")
	if s != "\x1b[3mhi\x1b[23m" {
		t.Errorf("Italic(hi) = %q", s)
	}
}

func TestReset(t *testing.T) {
	if Reset() != "\x1b[0m" {
		t.Errorf("Reset() = %q", Reset())
	}
}

// ---------------------------------------------------------------------------
// Text utility tests: VisibleLen
// ---------------------------------------------------------------------------

func TestVisibleLenPlainText(t *testing.T) {
	if n := VisibleLen("hello"); n != 5 {
		t.Errorf("VisibleLen(hello) = %d, want 5", n)
	}
}

func TestVisibleLenEmpty(t *testing.T) {
	if n := VisibleLen(""); n != 0 {
		t.Errorf("VisibleLen('') = %d, want 0", n)
	}
}

func TestVisibleLenANSI(t *testing.T) {
	s := "\x1b[31mred\x1b[0m"
	if n := VisibleLen(s); n != 3 {
		t.Errorf("VisibleLen(ANSI red) = %d, want 3", n)
	}
}

func TestVisibleLenWideChars(t *testing.T) {
	// Each CJK character is width 2.
	s := "\u4f60\u597d" // nihao
	n := VisibleLen(s)
	if n != 4 {
		t.Errorf("VisibleLen(CJK nihao) = %d, want 4", n)
	}
}

func TestVisibleLenMixedANSIAndWide(t *testing.T) {
	s := "\x1b[1m\u4f60\x1b[0m"
	n := VisibleLen(s)
	if n != 2 {
		t.Errorf("VisibleLen(ANSI+CJK) = %d, want 2", n)
	}
}

// ---------------------------------------------------------------------------
// Text utility tests: Truncate
// ---------------------------------------------------------------------------

func TestTruncateNoOp(t *testing.T) {
	s := "short"
	if r := Truncate(s, 10); r != s {
		t.Errorf("Truncate(short, 10) = %q, want %q", r, s)
	}
}

func TestTruncateCuts(t *testing.T) {
	s := "hello world"
	r := Truncate(s, 5)
	if VisibleLen(r) != 5 {
		t.Errorf("Truncate(hello world, 5) visible len = %d, want 5", VisibleLen(r))
	}
	if r != "hello" {
		t.Errorf("Truncate(hello world, 5) = %q, want %q", r, "hello")
	}
}

func TestTruncateZeroWidth(t *testing.T) {
	if r := Truncate("hello", 0); r != "" {
		t.Errorf("Truncate(hello, 0) = %q, want empty", r)
	}
}

func TestTruncatePreservesANSI(t *testing.T) {
	s := "\x1b[31mhello world\x1b[0m"
	r := Truncate(s, 5)
	vis := VisibleLen(r)
	if vis != 5 {
		t.Errorf("Truncate(ANSI, 5) visible len = %d, want 5", vis)
	}
	// The result should still contain the red escape.
	if !strings.Contains(r, "\x1b[31m") {
		t.Errorf("Truncate should preserve ANSI prefix, got %q", r)
	}
}

func TestTruncateWithTailEllipsis(t *testing.T) {
	s := "hello world"
	r := TruncateWithTail(s, 8, "...")
	vis := VisibleLen(r)
	if vis > 8 {
		t.Errorf("TruncateWithTail visible len = %d, want <= 8", vis)
	}
	if !strings.HasSuffix(r, "...") {
		t.Errorf("TruncateWithTail should end with '...', got %q", r)
	}
}

func TestTruncateWithTailShortString(t *testing.T) {
	s := "hi"
	r := TruncateWithTail(s, 10, "...")
	if r != "hi" {
		t.Errorf("TruncateWithTail(hi, 10, ...) = %q, want %q", r, "hi")
	}
}

// ---------------------------------------------------------------------------
// Text utility tests: Pad
// ---------------------------------------------------------------------------

func TestPadRightBasic(t *testing.T) {
	r := PadRight("hi", 5)
	if r != "hi   " {
		t.Errorf("PadRight(hi, 5) = %q, want %q", r, "hi   ")
	}
}

func TestPadRightNoOp(t *testing.T) {
	r := PadRight("hello", 3)
	if r != "hello" {
		t.Errorf("PadRight(hello, 3) should be unchanged, got %q", r)
	}
}

func TestPadRightWithANSI(t *testing.T) {
	s := "\x1b[31mhi\x1b[0m"
	r := PadRight(s, 5)
	vis := VisibleLen(r)
	if vis != 5 {
		t.Errorf("PadRight(ANSI, 5) visible len = %d, want 5", vis)
	}
}

func TestPadLeftBasic(t *testing.T) {
	r := PadLeft("hi", 5)
	if r != "   hi" {
		t.Errorf("PadLeft(hi, 5) = %q, want %q", r, "   hi")
	}
}

func TestPadLeftNoOp(t *testing.T) {
	r := PadLeft("hello", 3)
	if r != "hello" {
		t.Errorf("PadLeft(hello, 3) should be unchanged, got %q", r)
	}
}

func TestPadCenterBasic(t *testing.T) {
	r := PadCenter("hi", 6)
	if r != "  hi  " {
		t.Errorf("PadCenter(hi, 6) = %q, want %q", r, "  hi  ")
	}
}

func TestPadCenterOddPadding(t *testing.T) {
	r := PadCenter("hi", 7)
	// 7 - 2 = 5 total; left=2, right=3
	if r != "  hi   " {
		t.Errorf("PadCenter(hi, 7) = %q, want %q", r, "  hi   ")
	}
}

func TestPadCenterNoOp(t *testing.T) {
	r := PadCenter("hello", 3)
	if r != "hello" {
		t.Errorf("PadCenter(hello, 3) should be unchanged, got %q", r)
	}
}

// ---------------------------------------------------------------------------
// Text utility tests: Wrap
// ---------------------------------------------------------------------------

func TestWrapBasic(t *testing.T) {
	s := "hello world foo bar"
	lines := Wrap(s, 11)
	if len(lines) < 2 {
		t.Errorf("Wrap should produce multiple lines, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if VisibleLen(l) > 11 {
			t.Errorf("Wrap line too wide: %q (width %d)", l, VisibleLen(l))
		}
	}
}

func TestWrapZeroWidth(t *testing.T) {
	lines := Wrap("hello", 0)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("Wrap(hello, 0) = %v, want [hello]", lines)
	}
}

// ---------------------------------------------------------------------------
// Border style character tests
// ---------------------------------------------------------------------------

func TestBorderStyleSingleChars(t *testing.T) {
	chars := borderSets[BorderSingle]
	if chars.TopLeft != "\u250c" {
		t.Errorf("Single TopLeft = %q, want U+250C", chars.TopLeft)
	}
	if chars.TopRight != "\u2510" {
		t.Errorf("Single TopRight = %q, want U+2510", chars.TopRight)
	}
	if chars.BottomLeft != "\u2514" {
		t.Errorf("Single BottomLeft = %q, want U+2514", chars.BottomLeft)
	}
	if chars.BottomRight != "\u2518" {
		t.Errorf("Single BottomRight = %q, want U+2518", chars.BottomRight)
	}
	if chars.Horizontal != "\u2500" {
		t.Errorf("Single Horizontal = %q, want U+2500", chars.Horizontal)
	}
	if chars.Vertical != "\u2502" {
		t.Errorf("Single Vertical = %q, want U+2502", chars.Vertical)
	}
}

func TestBorderStyleDoubleChars(t *testing.T) {
	chars := borderSets[BorderDouble]
	if chars.TopLeft != "\u2554" {
		t.Errorf("Double TopLeft = %q, want U+2554", chars.TopLeft)
	}
	if chars.Horizontal != "\u2550" {
		t.Errorf("Double Horizontal = %q, want U+2550", chars.Horizontal)
	}
	if chars.Vertical != "\u2551" {
		t.Errorf("Double Vertical = %q, want U+2551", chars.Vertical)
	}
}

func TestBorderStyleRoundedChars(t *testing.T) {
	chars := borderSets[BorderRounded]
	if chars.TopLeft != "\u256d" {
		t.Errorf("Rounded TopLeft = %q, want U+256D", chars.TopLeft)
	}
	if chars.BottomRight != "\u256f" {
		t.Errorf("Rounded BottomRight = %q, want U+256F", chars.BottomRight)
	}
}

func TestBorderStyleHeavyChars(t *testing.T) {
	chars := borderSets[BorderHeavy]
	if chars.TopLeft != "\u250f" {
		t.Errorf("Heavy TopLeft = %q, want U+250F", chars.TopLeft)
	}
	if chars.Horizontal != "\u2501" {
		t.Errorf("Heavy Horizontal = %q, want U+2501", chars.Horizontal)
	}
	if chars.Vertical != "\u2503" {
		t.Errorf("Heavy Vertical = %q, want U+2503", chars.Vertical)
	}
}

func TestBorderStyleDashedChars(t *testing.T) {
	chars := borderSets[BorderDashed]
	if chars.Horizontal != "\u2504" {
		t.Errorf("Dashed Horizontal = %q, want U+2504", chars.Horizontal)
	}
	if chars.Vertical != "\u2506" {
		t.Errorf("Dashed Vertical = %q, want U+2506", chars.Vertical)
	}
}

// ---------------------------------------------------------------------------
// Box rendering tests
// ---------------------------------------------------------------------------

func TestRenderBoxMinimal(t *testing.T) {
	// Smallest possible box: 2x2 (just corners, no interior).
	style := BoxStyle{Border: BorderSingle}
	box := RenderBox("", 2, 2, style)
	lines := strings.Split(box, "\n")
	// Should have top border and bottom border.
	if len(lines) < 2 {
		t.Fatalf("2x2 box should have at least 2 lines, got %d", len(lines))
	}
	// Top border: TopLeft + TopRight.
	if !strings.Contains(lines[0], "\u250c") || !strings.Contains(lines[0], "\u2510") {
		t.Errorf("Top border missing corners: %q", lines[0])
	}
}

func TestRenderBoxTooSmall(t *testing.T) {
	style := BoxStyle{Border: BorderSingle}
	if box := RenderBox("", 1, 1, style); box != "" {
		t.Errorf("1x1 box should be empty, got %q", box)
	}
	if box := RenderBox("", 0, 5, style); box != "" {
		t.Errorf("0-width box should be empty, got %q", box)
	}
}

func TestRenderBoxSingleLine(t *testing.T) {
	style := BoxStyle{Border: BorderSingle}
	box := RenderBox("hi", 10, 3, style)
	lines := strings.Split(box, "\n")

	// 3 rows: top border, 1 content line, bottom border.
	// (The output will have a trailing empty string after the last \n on bottom border.)
	if len(lines) < 3 {
		t.Fatalf("10x3 box should have 3 lines, got %d: %v", len(lines), lines)
	}

	// Content line should contain "hi" padded to width 8 (10 - 2 borders).
	contentLine := lines[1]
	if !strings.Contains(contentLine, "hi") {
		t.Errorf("Content line should contain 'hi': %q", contentLine)
	}
}

func TestRenderBoxMultiLine(t *testing.T) {
	style := BoxStyle{Border: BorderRounded}
	content := "line1\nline2\nline3"
	box := RenderBox(content, 12, 5, style)
	lines := strings.Split(box, "\n")

	// 5 rows: top + 3 content + bottom.
	if len(lines) < 5 {
		t.Fatalf("12x5 box should have 5 lines, got %d: %v", len(lines), lines)
	}

	// Check each content line is present.
	for i, expected := range []string{"line1", "line2", "line3"} {
		if !strings.Contains(lines[i+1], expected) {
			t.Errorf("Line %d should contain %q, got %q", i+1, expected, lines[i+1])
		}
	}
}

func TestRenderBoxContentTruncation(t *testing.T) {
	style := BoxStyle{Border: BorderSingle}
	// Width 6 means interior width = 4. "toolong" (7 chars) should be truncated.
	box := RenderBox("toolong", 6, 3, style)
	lines := strings.Split(box, "\n")

	contentLine := lines[1]
	// Strip border chars to get interior.
	inner := contentLine[len("\u2502") : len(contentLine)-len("\u2502")]
	vis := VisibleLen(inner)
	if vis != 4 {
		t.Errorf("Truncated content visible width = %d, want 4 (inner=%q)", vis, inner)
	}
}

func TestRenderBoxContentPadding(t *testing.T) {
	// Short content should be right-padded.
	style := BoxStyle{Border: BorderSingle}
	box := RenderBox("ab", 8, 3, style)
	lines := strings.Split(box, "\n")
	contentLine := lines[1]
	// Interior width = 6. "ab" + 4 spaces.
	inner := contentLine[len("\u2502") : len(contentLine)-len("\u2502")]
	vis := VisibleLen(inner)
	if vis != 6 {
		t.Errorf("Padded content visible width = %d, want 6 (inner=%q)", vis, inner)
	}
}

func TestRenderBoxEmptyFill(t *testing.T) {
	// Height 5 with 1 content line => 2 empty interior lines filled.
	style := BoxStyle{Border: BorderSingle}
	box := RenderBox("only", 10, 5, style)
	lines := strings.Split(box, "\n")
	// top + 3 content rows + bottom = 5 visible lines.
	// lines[2] and lines[3] should be borders + spaces.
	for _, idx := range []int{2, 3} {
		if idx >= len(lines) {
			continue
		}
		line := lines[idx]
		// Should contain vertical borders and spaces only.
		stripped := strings.ReplaceAll(line, "\u2502", "")
		stripped = strings.TrimSpace(stripped)
		if stripped != "" {
			t.Errorf("Empty fill line %d should be blank, got %q", idx, line)
		}
	}
}

func TestRenderBoxWithPadding(t *testing.T) {
	style := BoxStyle{
		Border:  BorderSingle,
		Padding: NewPadding(1),
	}
	box := RenderBox("x", 10, 5, style)
	lines := strings.Split(box, "\n")

	// Height 5, borders take 2, padding top/bottom take 2, so 1 content row.
	// Width 10, borders take 2, padding left/right take 2, so 6 interior chars.
	if len(lines) < 5 {
		t.Fatalf("10x5 padded box should have 5 lines, got %d", len(lines))
	}

	// Lines[1] should be a padding row (border + spaces + border).
	paddingLine := lines[1]
	if !strings.HasPrefix(paddingLine, "\u2502") {
		t.Errorf("Padding row should start with vertical border: %q", paddingLine)
	}
}

// ---------------------------------------------------------------------------
// Title tests
// ---------------------------------------------------------------------------

func TestRenderBoxTitleLeft(t *testing.T) {
	style := BoxStyle{
		Border:     BorderSingle,
		Title:      "Test",
		TitleAlign: AlignLeft,
	}
	box := RenderBox("", 20, 3, style)
	lines := strings.Split(box, "\n")
	topBorder := lines[0]
	if !strings.Contains(topBorder, " Test ") {
		t.Errorf("Left-aligned title not found in top border: %q", topBorder)
	}
}

func TestRenderBoxTitleCenter(t *testing.T) {
	style := BoxStyle{
		Border:     BorderSingle,
		Title:      "Hi",
		TitleAlign: AlignCenter,
	}
	box := RenderBox("", 20, 3, style)
	lines := strings.Split(box, "\n")
	topBorder := lines[0]
	if !strings.Contains(topBorder, " Hi ") {
		t.Errorf("Centered title not found in top border: %q", topBorder)
	}
}

func TestRenderBoxTitleRight(t *testing.T) {
	style := BoxStyle{
		Border:     BorderSingle,
		Title:      "R",
		TitleAlign: AlignRight,
	}
	box := RenderBox("", 20, 3, style)
	lines := strings.Split(box, "\n")
	topBorder := lines[0]
	if !strings.Contains(topBorder, " R ") {
		t.Errorf("Right-aligned title not found in top border: %q", topBorder)
	}
}

func TestRenderBoxTitleTruncation(t *testing.T) {
	style := BoxStyle{
		Border: BorderSingle,
		Title:  "This is a very long title that should be truncated",
	}
	box := RenderBox("", 15, 3, style)
	lines := strings.Split(box, "\n")
	topBorder := lines[0]
	// The title should be truncated to fit within the border.
	vis := VisibleLen(topBorder)
	if vis != 15 {
		t.Errorf("Top border width = %d, want 15. Line: %q", vis, topBorder)
	}
}

// ---------------------------------------------------------------------------
// BorderNone tests
// ---------------------------------------------------------------------------

func TestRenderBoxNoBorder(t *testing.T) {
	style := BoxStyle{Border: BorderNone}
	box := RenderBox("hello", 10, 3, style)
	lines := strings.Split(box, "\n")
	// Should have 3 lines (trailing newlines produce extra empty).
	found := false
	for _, l := range lines {
		if strings.Contains(l, "hello") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("BorderNone box should contain 'hello': %q", box)
	}
	// Should NOT contain any box-drawing characters.
	for _, ch := range []string{"\u250c", "\u2500", "\u2502"} {
		if strings.Contains(box, ch) {
			t.Errorf("BorderNone box should not contain box-drawing char %q", ch)
		}
	}
}

// ---------------------------------------------------------------------------
// DefaultBoxStyle test
// ---------------------------------------------------------------------------

func TestDefaultBoxStyle(t *testing.T) {
	s := DefaultBoxStyle()
	if s.Border != BorderRounded {
		t.Errorf("DefaultBoxStyle border = %d, want BorderRounded", s.Border)
	}
	if s.Title != "" {
		t.Errorf("DefaultBoxStyle title = %q, want empty", s.Title)
	}
	if s.Padding != (Padding{}) {
		t.Errorf("DefaultBoxStyle padding = %+v, want zero", s.Padding)
	}
}

// ---------------------------------------------------------------------------
// Box with colors test
// ---------------------------------------------------------------------------

func TestRenderBoxWithColor(t *testing.T) {
	style := BoxStyle{
		Border: BorderSingle,
		FG:     "#ff0000",
	}
	box := RenderBox("test", 10, 3, style)
	// Should contain the foreground color escape.
	if !strings.Contains(box, "\x1b[38;2;255;0;0m") {
		t.Errorf("Colored box should contain fg escape: %q", box)
	}
	// Should contain reset.
	if !strings.Contains(box, "\x1b[0m") {
		t.Errorf("Colored box should contain reset: %q", box)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestVisibleLenWidthOne(t *testing.T) {
	if n := VisibleLen("x"); n != 1 {
		t.Errorf("VisibleLen(x) = %d, want 1", n)
	}
}

func TestTruncateWidthOne(t *testing.T) {
	r := Truncate("hello", 1)
	if VisibleLen(r) != 1 {
		t.Errorf("Truncate(hello, 1) visible len = %d, want 1", VisibleLen(r))
	}
}

func TestPadRightWidthZero(t *testing.T) {
	r := PadRight("hi", 0)
	if r != "hi" {
		t.Errorf("PadRight(hi, 0) = %q, want %q", r, "hi")
	}
}

func TestPadLeftWidthZero(t *testing.T) {
	r := PadLeft("hi", 0)
	if r != "hi" {
		t.Errorf("PadLeft(hi, 0) = %q, want %q", r, "hi")
	}
}

func TestPadCenterWidthZero(t *testing.T) {
	r := PadCenter("hi", 0)
	if r != "hi" {
		t.Errorf("PadCenter(hi, 0) = %q, want %q", r, "hi")
	}
}

func TestPadRightEmptyString(t *testing.T) {
	r := PadRight("", 5)
	if r != "     " {
		t.Errorf("PadRight('', 5) = %q, want 5 spaces", r)
	}
}

func TestRenderBoxAllBorderStyles(t *testing.T) {
	styles := []BorderStyle{BorderSingle, BorderDouble, BorderRounded, BorderHeavy, BorderDashed}
	for _, bs := range styles {
		box := RenderBox("ok", 10, 3, BoxStyle{Border: bs})
		if box == "" {
			t.Errorf("RenderBox with border style %d should not be empty", bs)
		}
		lines := strings.Split(box, "\n")
		if len(lines) < 3 {
			t.Errorf("Border style %d: expected at least 3 lines, got %d", bs, len(lines))
		}
	}
}

func TestFitLineExactWidth(t *testing.T) {
	r := fitLine("abcde", 5)
	if r != "abcde" {
		t.Errorf("fitLine(abcde, 5) = %q, want abcde", r)
	}
}

func TestFitLineZeroWidth(t *testing.T) {
	r := fitLine("hello", 0)
	if r != "" {
		t.Errorf("fitLine(hello, 0) = %q, want empty", r)
	}
}
