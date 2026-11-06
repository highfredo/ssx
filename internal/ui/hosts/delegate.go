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
	"github.com/highfredo/ssx/internal/ui/list"
)

// hostDelegate wraps DefaultDelegate and appends tag chips to the title line
// after the delegate has already applied its styling, avoiding interference
// with lipgloss.StyleRunes during filter highlighting.
type hostDelegate struct {
	blist.DefaultDelegate
	tagColors map[string]string // default colour per tag name, from the app config
}

// Compact implements list.Compactable.
func (d hostDelegate) Compact() blist.ItemDelegate {
	d.DefaultDelegate = list.CompactDelegate(d.DefaultDelegate)
	return d
}

func (d hostDelegate) Render(w io.Writer, m blist.Model, index int, listItem blist.Item) {
	it, ok := listItem.(item)
	if !ok || it.host == nil {
		d.DefaultDelegate.Render(w, m, index, listItem)
		return
	}

	// Render into a buffer first so we can decorate the lines.
	var buf strings.Builder
	d.DefaultDelegate.Render(&buf, m, index, listItem)
	lines := strings.SplitN(buf.String(), "\n", 2)

	h := it.host
	if len(h.Tags) > 0 {
		lines[0] += " " + d.renderTags(h.Tags, m.FilterInput.Value())
	}
	if h.IsAdHoc() && len(lines) > 1 {
		lines[1] += "  " + styles.MutedStyle.Render("· enter to connect")
	}
	for i := range lines {
		lines[i] = list.TruncateLine(lines[i], m.Width())
	}
	fmt.Fprint(w, strings.Join(lines, "\n"))
}

// renderTags draws the tag chips, underlining filter matches.
func (d hostDelegate) renderTags(tags []ssh.Tag, filterTerm string) string {
	renderedTags := make([]string, len(tags))
	for j, tag := range tags {
		chipStyle := d.tagChipStyle(tag)
		unmatched := chipStyle.Inline(true)
		matched := unmatched.Inherit(d.Styles.FilterMatch)

		chipText := tag.Name
		if filterTerm != "" {
			if ranks := list.Filter(filterTerm, []string{tag.Name}); len(ranks) > 0 && len(ranks[0].MatchedIndexes) > 0 {
				chipText = lipgloss.StyleRunes(tag.Name, ranks[0].MatchedIndexes, matched, unmatched)
			}
		}
		renderedTags[j] = chipStyle.Render(chipText)
	}
	return strings.Join(renderedTags, " ")
}

// tagChipStyle styles a tag chip with its inline colour, the configured
// default for that tag name, or the theme default.
func (d hostDelegate) tagChipStyle(tag ssh.Tag) lipgloss.Style {
	c := tag.Color
	if c == "" {
		c = d.tagColors[tag.Name]
	}
	if c == "" {
		return styles.TagChipStyle
	}
	return styles.ChipStyle(lipgloss.Color(colors.Resolve(c)))
}
