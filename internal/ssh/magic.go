package ssh

import (
	"strings"

	gossh "github.com/kevinburke/ssh_config"
)

// Magic comments store ssx metadata inside ~/.ssh/config using the form
// "#>key:value" on a line of its own or at the end of a directive, e.g.
//
//	#>tags:dev,aws:orange
//	#>password:secret
//	#>passwordCommand:bw get password myhost
//	#>hidden:true
const magicPrefix = ">"

// nodeComment returns the trimmed comment attached to a node, or "" when the
// node type carries no comment.
func nodeComment(node gossh.Node) string {
	switch n := node.(type) {
	case *gossh.Empty:
		return strings.TrimSpace(n.Comment)
	case *gossh.KV:
		return strings.TrimSpace(n.Comment)
	}
	return ""
}

// magicValue returns the value of the first "#>key:" comment in nodes, or "".
func magicValue(nodes []gossh.Node, key string) string {
	prefix := magicPrefix + key + ":"
	for _, node := range nodes {
		if after, ok := strings.CutPrefix(nodeComment(node), prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// extractTags merges every "#>tags:" comment in nodes. Each entry has the form
// name[:color]. When a tag name repeats (e.g. a host overriding "Host *"), the
// first occurrence wins, preserving the host-specific color. Tags without an
// inline color have an empty Color; the UI decides their default.
func extractTags(nodes []gossh.Node) []Tag {
	prefix := magicPrefix + "tags:"
	seen := make(map[string]bool)
	var tags []Tag

	for _, node := range nodes {
		after, ok := strings.CutPrefix(nodeComment(node), prefix)
		if !ok {
			continue
		}
		for raw := range strings.SplitSeq(after, ",") {
			name, color, _ := strings.Cut(strings.TrimSpace(raw), ":")
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			tags = append(tags, Tag{Name: name, Color: strings.TrimSpace(color)})
		}
	}
	return tags
}
