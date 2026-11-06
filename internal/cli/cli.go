// Package cli parses command-line arguments and runs the non-interactive
// commands of ssx (direct connect, tunnel open, shell completion).
package cli

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/highfredo/ssx/internal/ssh"
	kongcompletion "github.com/jotaen/kong-completion"
	"github.com/posener/complete"
)

// BuildInfo carries the version metadata injected at build time.
type BuildInfo struct {
	Version string
	Date    string
	Commit  string
}

// VersionFlag prints build information and exits.
type VersionFlag bool

// BeforeReset implements kong's hook so "--version" short-circuits parsing.
func (v VersionFlag) BeforeReset(app *kong.Kong, vars kong.Vars) error {
	fmt.Println("Version: " + vars["version"])
	fmt.Println("Date: " + vars["date"])
	fmt.Println("Commit: " + vars["commit"])
	app.Exit(0)
	return nil
}

// ConnectCmd connects to a host or opens one of its tunnels without the TUI.
type ConnectCmd struct {
	Host   string `arg:"" optional:"" completion-predictor:"host" help:"Host alias or user@host[:port]."`
	Tunnel string `arg:"" optional:"" completion-predictor:"tunnel" help:"Local port or localPort:remoteHost:remotePort."`
}

// CLI is the root command tree.
type CLI struct {
	Connect    ConnectCmd                `cmd:"" default:"withargs" completion-enabled:"false" help:"Connect to a host or open a tunnel."`
	Completion kongcompletion.Completion `cmd:"" completion-enabled:"false" help:"Outputs shell code for initialising tab completions."`
	Version    VersionFlag               `short:"v" name:"version" help:"Print version information."`
}

// Parse processes command-line arguments and completions. It returns
// handled=true when a CLI action was executed, or handled=false when the TUI
// should start.
func Parse(info BuildInfo) (handled bool, err error) {
	var c CLI
	parser := kong.Must(&c,
		kong.Name("ssx"),
		kong.Description("Terminal User Interface for managing SSH connections and tunnels."),
		kong.UsageOnError(),
		kong.Vars{
			"version": info.Version,
			"date":    info.Date,
			"commit":  info.Commit,
		},
	)

	// Attach positional arguments to the root node only during completion
	// so `ssx <TAB>` and `ssx <host> <TAB>` complete directly without a subcommand.
	if len(parser.Model.Children) > 0 && parser.Model.Children[0].Name == "connect" {
		parser.Model.Positional = parser.Model.Children[0].Positional
	}
	kongcompletion.Register(parser,
		kongcompletion.WithPredictor("host", complete.PredictFunc(predictHosts)),
		kongcompletion.WithPredictor("tunnel", complete.PredictFunc(predictTunnels)),
	)
	parser.Model.Positional = nil

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		return true, err
	}

	if ctx.Command() == "completion" || ctx.Command() == "completion <shell>" {
		return true, ctx.Run()
	}
	if c.Connect.Host == "" {
		return false, nil
	}
	return true, c.Connect.Run()
}

// Run executes the connect command.
func (c *ConnectCmd) Run() error {
	host, err := c.resolveHost()
	if err != nil {
		return err
	}
	if c.Tunnel == "" {
		return runShell(host)
	}
	tunnel := findTunnel(host, c.Tunnel)
	if tunnel == nil {
		return fmt.Errorf("tunnel %q not found for host %q", c.Tunnel, c.Host)
	}
	return openTunnel(host, *tunnel)
}

// resolveHost looks the host up in the ssh config. Only when no tunnel is
// requested does it accept an ad-hoc user@host[:port] connection string.
func (c *ConnectCmd) resolveHost() (*ssh.HostConfig, error) {
	hosts, err := ssh.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading ssh config: %w", err)
	}
	host := hosts.Get(c.Host)
	if host == nil && c.Tunnel == "" {
		host = ssh.ParseConnection(c.Host)
	}
	if host == nil {
		return nil, fmt.Errorf("host %q not found", c.Host)
	}
	return host, nil
}

func runShell(host *ssh.HostConfig) error {
	cmd := ssh.Shell(host)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findTunnel resolves spec against the host's configured tunnels by local
// port, falling back to parsing it as a quick-tunnel specification.
func findTunnel(host *ssh.HostConfig, spec string) *ssh.Tunnel {
	for i := range host.Tunnels {
		if host.Tunnels[i].LocalPort == spec {
			return &host.Tunnels[i]
		}
	}
	return ssh.ParseTunnel(host.Name, spec)
}

func openTunnel(host *ssh.HostConfig, tunnel ssh.Tunnel) error {
	status, err := ssh.GetTunnelStatus(host, tunnel)
	if err != nil {
		return err
	}
	switch status.State {
	case ssh.TunnelOpened:
		fmt.Printf("Tunnel %s:%s is already open\n", host.Name, tunnel.LocalPort)
		return nil
	case ssh.TunnelPortInUse:
		return fmt.Errorf("port %s is already in use (pid %d, %s)", tunnel.LocalPort, status.PID, status.Owner)
	}

	status, err = ssh.OpenTunnel(host, tunnel)
	if err != nil {
		return err
	}
	if status.State != ssh.TunnelOpened {
		return fmt.Errorf("failed to open tunnel %s:%s", host.Name, tunnel.LocalPort)
	}
	fmt.Printf("Tunnel %s:%s opened\n", host.Name, tunnel.LocalPort)
	return nil
}

func predictHosts(complete.Args) []string {
	hosts, err := ssh.LoadConfig()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	return names
}

func predictTunnels(args complete.Args) []string {
	hosts, err := ssh.LoadConfig()
	if err != nil {
		return nil
	}
	host := hosts.Get(args.LastCompleted)
	if host == nil {
		return nil
	}
	ports := make([]string, 0, len(host.Tunnels))
	for _, t := range host.Tunnels {
		ports = append(ports, t.LocalPort)
	}
	return ports
}
