package layout

// SplitVertical is a convenience function that splits area top-to-bottom
// according to the given constraints.
func SplitVertical(area Rect, constraints ...Constraint) []Rect {
	return NewLayout(Vertical, constraints...).Split(area)
}

// SplitHorizontal is a convenience function that splits area left-to-right
// according to the given constraints.
func SplitHorizontal(area Rect, constraints ...Constraint) []Rect {
	return NewLayout(Horizontal, constraints...).Split(area)
}

// LayoutBuilder provides a fluent API for constructing layouts.
type LayoutBuilder struct {
	layout *Layout
}

// NewLayoutBuilder creates a new builder with sensible defaults.
func NewLayoutBuilder() *LayoutBuilder {
	return &LayoutBuilder{
		layout: &Layout{
			direction: Horizontal,
		},
	}
}

// Direction sets the split direction.
func (b *LayoutBuilder) Direction(d Direction) *LayoutBuilder {
	b.layout.direction = d
	return b
}

// Constraints sets the constraint list.
func (b *LayoutBuilder) Constraints(cs ...Constraint) *LayoutBuilder {
	b.layout.constraints = cs
	return b
}

// Flex sets the flex mode.
func (b *LayoutBuilder) Flex(f Flex) *LayoutBuilder {
	b.layout.flex = f
	return b
}

// Spacing sets the gap between items.
func (b *LayoutBuilder) Spacing(s int) *LayoutBuilder {
	b.layout.spacing = s
	return b
}

// Margin sets the outer margin.
func (b *LayoutBuilder) Margin(m int) *LayoutBuilder {
	b.layout.margin = m
	return b
}

// Split runs the solver on the given area.
func (b *LayoutBuilder) Split(area Rect) []Rect {
	return b.layout.Split(area)
}

// Build returns the underlying Layout for reuse.
func (b *LayoutBuilder) Build() *Layout {
	return b.layout
}
