package hosts

import (
	"fmt"
	"io"
	"strings"

	blist "charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/colors"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
)

// hostDelegate wraps DefaultDelegate and appends tag chips to the title line
// after the delegate has already applied its styling, avoiding interference
// with lipgloss.StyleRunes during filter highlighting.
type hostDelegate struct {
	blist.DefaultDelegate
}

func (d hostDelegate) Render(w io.Writer, m blist.Model, index int, listItem blist.Item) {
	it, ok := listItem.(item)
	if !ok || len(it.host.Tags) == 0 {
		d.DefaultDelegate.Render(w, m, index, listItem)
		return
	}

	// Render into a buffer first so we can inject tags into the title line.
	var buf strings.Builder
	d.DefaultDelegate.Render(&buf, m, index, listItem)
	rendered := buf.String()

	h := it.host
	filterTerm := m.FilterInput.Value()

	renderedTags := make([]string, len(h.Tags))
	for j, tag := range h.Tags {
		chipStyle := tagChipStyle(tag)
		unmatched := chipStyle.Inline(true)
		matched := unmatched.Inherit(d.Styles.FilterMatch)

		chipText := tag.Name
		if filterTerm != "" {
			if ranks := blist.DefaultFilter(filterTerm, []string{tag.Name}); len(ranks) > 0 && len(ranks[0].MatchedIndexes) > 0 {
				chipText = lipgloss.StyleRunes(tag.Name, ranks[0].MatchedIndexes, matched, unmatched)
			}
		}
		renderedTags[j] = chipStyle.Render(chipText)
	}
	tagStr := " " + strings.Join(renderedTags, " ")

	// Insert tags at the end of the first line (title line).
	if nl := strings.Index(rendered, "\n"); nl >= 0 {
		fmt.Fprint(w, rendered[:nl]+tagStr+rendered[nl:])
	} else {
		fmt.Fprint(w, rendered+tagStr)
	}
}

func tagChipStyle(tag ssh.Tag) lipgloss.Style {
	if tag.Color == "" {
		return styles.TagChipStyle
	}
	c := colors.Resolve(tag.Color)
	return styles.TagChipStyle.
		Background(lipgloss.Color(c)).
		Foreground(colors.TintedForeground(c))
}
