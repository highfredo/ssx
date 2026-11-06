package info

// sectionDef groups ssh -G keys under a themed title.
type sectionDef struct {
	title string
	keys  []string // keys in lowercase as printed by ssh -G
}

var sshSections = []sectionDef{
	{
		title: "🪄 ssx",
		keys:  []string{"tags", "password", "passwordCommand"},
	},
	{
		title: "🔌 Connection",
		keys: []string{
			"hostname", "port", "user", "addressfamily",
			"connecttimeout", "connectionattempts",
			"serveralivecountmax", "serveraliveinterval", "tcpkeepalive",
		},
	},
	{
		title: "⟳ Forwarding",
		keys: []string{
			"localforward", "remoteforward", "dynamicforward",
			"forwardagent", "forwardx11", "forwardx11timeout", "forwardx11trusted",
			"gatewayports", "clearallforwardings", "permitremoteopen", "permittunnel",
		},
	},
	{
		title: "🔀 Proxy",
		keys:  []string{"proxycommand", "proxyjump"},
	},
	{
		title: "🔐 Authentication",
		keys: []string{
			"identityfile", "identityagent", "certificate",
			"preferredauthentications", "pubkeyauthentication",
			"passwordauthentication", "kbdinteractiveauthentication",
			"hostbasedauthentication", "gssapiauthentication",
			"numberofpasswordprompts",
		},
	},
	{
		title: "🔒 Cryptography",
		keys: []string{
			"ciphers", "macs", "hostkeyalgorithms", "kexalgorithms",
			"pubkeyacceptedalgorithms", "casignaturealgorithms",
			"fingerprinthash", "hostbasedacceptedalgorithms",
		},
	},
	{
		title: "🛡️ Host Verification",
		keys: []string{
			"stricthostkeychecking", "userknownhostsfile", "globalknownhostsfile",
			"checkhostip", "verifyhostkeydns",
		},
	},
	{
		title: "⚡ Multiplexing",
		keys:  []string{"controlmaster", "controlpath", "controlpersist"},
	},
	{
		title: "🖥️ Session",
		keys: []string{
			"requesttty", "sessiontype", "permittylocalcommand", "localcommand",
			"sendenv", "setenv", "remotecommand",
		},
	},
}
