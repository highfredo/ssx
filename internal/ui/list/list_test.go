package list

import (
	"testing"

	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/ui/base"
)

type testItem string

func (t testItem) Title() string       { return string(t) }
func (t testItem) Description() string { return "" }
func (t testItem) FilterValue() string { return string(t) }

func TestVirtualItemSearchAndBackspace(t *testing.T) {
	items := []blist.Item{testItem("alpha"), testItem("beta")}
	l := New(items, NewDefaultDelegate(), base.NewKeybindings(appconfig.KeybindingsConfig{}).Navigation)
	l.SetSize(80, 24)
	l.SetVirtualItemBuilder(func(q string) blist.Item {
		if q != "" {
			return testItem("virtual:" + q)
		}
		return nil
	})

	// Type 'x' (matches no real item, adds virtual item)
	cmd := l.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			l.Update(msg)
		}
	}
	_ = l.View()

	// Press Backspace (removes virtual item, resets filter)
	cmd = l.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			l.Update(msg)
		}
	}
	_ = l.View()

	if len(l.Items()) != 2 {
		t.Fatalf("expected 2 items after backspace, got %d", len(l.Items()))
	}
}

func TestPromoteVirtualItemClearsSearchAndSelectsLast(t *testing.T) {
	items := []blist.Item{testItem("alpha"), testItem("beta")}
	l := New(items, NewDefaultDelegate(), base.NewKeybindings(appconfig.KeybindingsConfig{}).Navigation)
	l.SetSize(80, 24)
	l.SetVirtualItemBuilder(func(q string) blist.Item {
		if q != "" {
			return testItem("virtual:" + q)
		}
		return nil
	})

	l.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if _, idx := l.VirtualItem(); idx != 2 {
		t.Fatalf("expected virtual item at index 2, got %d", idx)
	}

	l.PromoteVirtualItem(testItem("promoted"))

	if got := l.SearchValue(); got != "" {
		t.Errorf("expected search cleared, got %q", got)
	}
	if vItem, _ := l.VirtualItem(); vItem != nil {
		t.Error("expected no virtual item after promotion")
	}
	if n := len(l.VisibleItems()); n != 3 {
		t.Fatalf("expected 3 visible items after promotion, got %d", n)
	}
	if got := l.SelectedItem(); got != testItem("promoted") {
		t.Errorf("expected promoted item selected, got %v", got)
	}
}

func TestByteToRuneIndexes(t *testing.T) {
	// "→" is 3 bytes: bytes 0-2 → rune 0, byte 3 → rune 1, byte 4 → rune 2.
	s := "→ab"
	got := byteToRuneIndexes(s, []int{3, 4})
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected [1 2], got %v", got)
	}
	if got := byteToRuneIndexes("abc", []int{0, 2}); len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("ASCII must be identity, got %v", got)
	}
}
