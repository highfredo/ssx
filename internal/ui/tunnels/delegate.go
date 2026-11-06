package tunnels

import (
	"fmt"
	"io"
	"strings"

	blist "charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/list"
)

// tunnelDelegate wraps DefaultDelegate and decorates each row: the status
// badge is appended to the title line, and the port owner / last error are
// appended to the description line. Decorations are added
// after the default rendering so filter highlighting is not disturbed.
type tunnelDelegate struct {
	blist.DefaultDelegate
	showHost bool // append the owning host to each row (open-tunnels page)
}

func newDelegate(showHost bool) tunnelDelegate {
	return tunnelDelegate{DefaultDelegate: list.NewDefaultDelegate(), showHost: showHost}
}

// Compact implements list.Compactable.
func (d tunnelDelegate) Compact() blist.ItemDelegate {
	d.DefaultDelegate = list.CompactDelegate(d.DefaultDelegate)
	return d
}

func (d tunnelDelegate) Render(w io.Writer, m blist.Model, index int, listItem blist.Item) {
	it, ok := listItem.(item)
	if !ok {
		d.DefaultDelegate.Render(w, m, index, listItem)
		return
	}

	var buf strings.Builder
	d.DefaultDelegate.Render(&buf, m, index, listItem)
	lines := strings.SplitN(buf.String(), "\n", 2)

	if it.quick {
		lines[0] = strings.Replace(lines[0], it.Title(), "⚡"+it.Title(), 1)
	}
	// Pad the title so status badges line up in a column.
	if pad := titleColumnWidth(m) - lipgloss.Width(lines[0]); pad > 0 {
		lines[0] += strings.Repeat(" ", pad)
	}
	lines[0] += "  " + statusBadge(it.status)
	if d.showHost && it.tunnel.Host != "" {
		lines[0] += "  " + hostLabel(it.tunnel.Host)
	}
	if len(lines) > 1 {
		if extra := runtimeDetails(it.status); extra != "" {
			lines[1] += "  " + extra
		}
	}
	for i := range lines {
		lines[i] = list.TruncateLine(lines[i], m.Width())
	}
	fmt.Fprint(w, strings.Join(lines, "\n"))
}

// titleColumnWidth returns the rendered width of the widest title among the
// visible items, including the cursor prefix drawn by the default delegate.
func titleColumnWidth(m blist.Model) int {
	const cursorPrefix = 2
	w := 0
	for _, li := range m.VisibleItems() {
		if it, ok := li.(item); ok {
			tw := lipgloss.Width(it.Title())
			if it.quick {
				tw++ // leading ⚡
			}
			w = max(w, tw)
		}
	}
	return w + cursorPrefix
}

func statusBadge(status ssh.TunnelStatus) string {
	switch status.State {
	case ssh.TunnelOpened:
		return styles.StateOpenStyle.Render("● open")
	case ssh.TunnelOpening:
		return styles.StateOpeningStyle.Render("◐ opening")
	case ssh.TunnelPortInUse:
		return styles.StateInUseStyle.Render("⚠ in use")
	case ssh.TunnelClosed:
		return styles.StateClosedStyle.Render("○ closed")
	default:
		return styles.StateClosedStyle.Render("◌ unknown")
	}
}

// hostLabel renders the alias of the host a tunnel goes through.
func hostLabel(host string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render("@" + host)
}

// runtimeDetails describes who holds the port and the last error, if any.
func runtimeDetails(status ssh.TunnelStatus) string {
	var parts []string
	if status.State == ssh.TunnelPortInUse && status.Owner != "" {
		owner := status.Owner
		if status.PID > 0 {
			owner = fmt.Sprintf("%s (pid %d)", owner, status.PID)
		}
		parts = append(parts, styles.MutedStyle.Render("· "+owner))
	}
	if status.Err != nil {
		parts = append(parts, styles.ErrorStyle.Render("✖ "+status.Err.Error()))
	}
	return strings.Join(parts, "  ")
}
