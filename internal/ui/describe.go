package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// describeHorizontalPadding is the horizontal padding value for the overlay border.
	describeHorizontalPadding = 2

	// describeInputChrome is the total horizontal space consumed by the overlay's
	// border (2) and padding (4) on each side: (1+2)*2 = 12.
	describeInputChrome = 12

	// minDescribeInputWidth is the floor width for the text input field.
	minDescribeInputWidth = 20
)

// DescribeInput is a text input overlay for editing change descriptions.
type DescribeInput struct {
	input    textinput.Model
	changeID string
	width    int
	height   int

	// Key bindings
	submit key.Binding
	cancel key.Binding

	// Styles
	borderStyle lipgloss.Style
	titleStyle  lipgloss.Style
	hintStyle   lipgloss.Style
}

// NewDescribeInput creates a new describe input overlay.
func NewDescribeInput() *DescribeInput {
	input := textinput.New()
	input.Placeholder = "Enter description..."
	input.CharLimit = 256
	input.Focus()

	return &DescribeInput{
		input: input,
		submit: key.NewBinding(
			key.WithKeys("enter"),
		),
		cancel: key.NewBinding(
			key.WithKeys("esc"),
		),
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, describeHorizontalPadding),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		hintStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// SetSize sets the available size for the overlay.
func (d *DescribeInput) SetSize(width, height int) {
	d.width = width
	d.height = height

	// Set input width to fit within the modal
	// Account for border (2) + padding (4) on each side
	inputWidth := width - describeInputChrome
	if inputWidth < minDescribeInputWidth {
		inputWidth = minDescribeInputWidth
	}

	d.input.SetWidth(inputWidth)
}

// SetChangeID sets the change ID being edited.
func (d *DescribeInput) SetChangeID(changeID string) {
	d.changeID = changeID
}

// SetValue sets the current description text.
func (d *DescribeInput) SetValue(value string) {
	d.input.SetValue(value)
	// Move cursor to end
	d.input.CursorEnd()
}

// Value returns the current input value.
func (d *DescribeInput) Value() string {
	return d.input.Value()
}

// ChangeID returns the change ID being edited.
func (d *DescribeInput) ChangeID() string {
	return d.changeID
}

// Focus sets focus on the text input.
func (d *DescribeInput) Focus() tea.Cmd {
	return d.input.Focus()
}

// DescribeSubmitMsg is sent when the user submits the description.
type DescribeSubmitMsg struct {
	ChangeID    string
	Description string
}

// DescribeCancelMsg is sent when the user cancels editing.
type DescribeCancelMsg struct{}

// Update handles input messages.
func (d *DescribeInput) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, d.submit) {
			return func() tea.Msg {
				return DescribeSubmitMsg{
					ChangeID:    d.changeID,
					Description: d.input.Value(),
				}
			}
		}

		if key.Matches(msg, d.cancel) {
			return func() tea.Msg {
				return DescribeCancelMsg{}
			}
		}
	}

	// Forward to text input
	var cmd tea.Cmd

	d.input, cmd = d.input.Update(msg)

	return cmd
}

// View renders the describe input overlay.
func (d *DescribeInput) View() string {
	title := d.titleStyle.Render("Describe: " + d.changeID)
	hint := d.hintStyle.Render("⏎ save • ⎋ cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		d.input.View(),
		"",
		hint,
	)

	return d.borderStyle.Render(content)
}

// Width returns the rendered width of the overlay.
func (d *DescribeInput) Width() int {
	return lipgloss.Width(d.View())
}

// Height returns the rendered height of the overlay.
func (d *DescribeInput) Height() int {
	return lipgloss.Height(d.View())
}
