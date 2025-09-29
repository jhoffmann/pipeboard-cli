// Package ui provides status bar component for displaying application context
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// statusBarColors defines the color scheme for the status bar sections
type statusBarColors struct {
	leftBg     lipgloss.Color
	leftFg     lipgloss.Color
	middleBg   lipgloss.Color
	middleFg   lipgloss.Color
	rightBg    lipgloss.Color
	rightFg    lipgloss.Color
	normalText lipgloss.Color
	boldText   lipgloss.Color
}

// defaultStatusColors returns the default color scheme for the status bar
func defaultStatusColors() statusBarColors {
	return statusBarColors{
		leftBg:     lipgloss.Color("#2d3142"), // Dark gray
		leftFg:     lipgloss.Color("#ffffff"), // White
		middleBg:   lipgloss.Color("#4f5d75"), // Medium gray
		middleFg:   lipgloss.Color("#ffffff"), // White
		rightBg:    lipgloss.Color("#9c4dcc"), // Purple
		rightFg:    lipgloss.Color("#ffffff"), // White
		normalText: lipgloss.Color("#bfc0c0"), // Light gray for normal text
		boldText:   lipgloss.Color("#ffffff"), // White for bold text
	}
}

// renderStatusBar creates a status bar with three sections: program info, profile, and region
func renderStatusBar(width int, version, profile, region string) string {
	colors := defaultStatusColors()

	// Left section: "pipeboard v{version}"
	leftStyle := lipgloss.NewStyle().
		Background(colors.leftBg).
		Foreground(colors.leftFg).
		Padding(0, 2)

	// Create styles with the same background as leftStyle
	normalStyle := lipgloss.NewStyle().
		Background(colors.leftBg).
		Foreground(colors.normalText)
	boldStyle := lipgloss.NewStyle().
		Background(colors.leftBg).
		Foreground(colors.boldText).
		Bold(true)

	pipeText := normalStyle.Render("pipe")
	boardText := boldStyle.Render("board")
	versionText := normalStyle.Render(" " + version)
	leftContent := pipeText + boardText + versionText
	leftContent = leftStyle.Render(leftContent)

	// Middle section: AWS profile
	middleStyle := lipgloss.NewStyle().
		Background(colors.middleBg).
		Foreground(colors.middleFg).
		Padding(0, 2)

	profileText := profile
	if profile == "" {
		profileText = "default"
	}
	middleContent := middleStyle.Render(profileText)

	// Right section: AWS region
	rightStyle := lipgloss.NewStyle().
		Background(colors.rightBg).
		Foreground(colors.rightFg).
		Padding(0, 2).
		Bold(true)

	regionText := region
	if region == "" {
		regionText = "unknown"
	}
	rightContent := rightStyle.Render(regionText)

	// Calculate remaining space for flexible middle section
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	remainingWidth := width - leftWidth - rightWidth

	// If we have extra space, expand the middle section
	if remainingWidth > 0 {
		middleStyle = middleStyle.Width(remainingWidth)
		middleContent = middleStyle.Render(profileText)
	}

	// Join all sections horizontally
	statusBar := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, middleContent, rightContent)

	// Ensure the status bar fills the full width
	if lipgloss.Width(statusBar) < width {
		statusBar = lipgloss.NewStyle().Width(width).Render(statusBar)
	}

	return statusBar
}
