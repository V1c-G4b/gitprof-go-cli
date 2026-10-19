package setup

import (
	"fmt"
	"gitprof-go-cli/internal/profile"
	"strings"
)

func needsHostBlock(p profile.Profile) bool {
	h := strings.TrimSpace(p.SSHHost)
	return h != "" && !strings.EqualFold(h, "github.com")
}

func RenderSSHFragment(profiles []profile.Profile) (string, []string) {
	var b strings.Builder
	var warnings []string
	seen := map[string]string{}
	for _, p := range profiles {
		if !needsHostBlock(p) {
			continue
		}
		host := strings.TrimSpace(p.SSHHost)
		key := strings.ToLower(host)
		if other, dup := seen[key]; dup {
			warnings = append(warnings, fmt.Sprintf("perfis %q e %q usam o mesmo alias %q; só o primeiro entra no ssh config", other, p.Name, host))
			continue
		}
		if strings.TrimSpace(p.Key) == "" {
			warnings = append(warnings, fmt.Sprintf("perfil %q usa o alias %q mas não tem chave; rode gitprof profile key -name %s -key <arquivo>", p.Name, host, p.Name))
			continue
		}
		seen[key] = p.Name
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Host %s\n", host)
		b.WriteString("  HostName github.com\n")
		b.WriteString("  User git\n")
		fmt.Fprintf(&b, "  IdentityFile %s\n", sshQuote(p.Key))
		b.WriteString("  IdentitiesOnly yes\n")
	}
	return b.String(), warnings
}

func sshQuote(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

func isIncludeLine(line string) bool {
	fields := strings.Fields(line)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Include") {
		return false
	}
	arg := strings.Trim(fields[1], `"`)
	return arg == "~/.ssh/config.d/gitprof" || arg == "config.d/gitprof"
}

func isBlockStart(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	kw := strings.ToLower(strings.SplitN(fields[0], "=", 2)[0])
	return kw == "host" || kw == "match"
}

func EnsureInclude(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	count := 0
	beforeBlocks := false
	inBlock := false
	for _, l := range lines {
		if isBlockStart(l) {
			inBlock = true
		}
		if isIncludeLine(l) {
			count++
			if !inBlock {
				beforeBlocks = true
			}
		}
	}
	if count == 1 && beforeBlocks {
		return content
	}

	var b strings.Builder
	b.WriteString(IncludeLine + "\n")
	if count == 0 && len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		b.WriteString("\n")
	}
	for _, l := range lines {
		if isIncludeLine(l) {
			continue
		}
		b.WriteString(l)
	}
	out := b.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}
