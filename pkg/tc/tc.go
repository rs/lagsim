package tc

import (
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	DryRun  bool
	Verbose bool
}

func (r *Runner) Run(args ...string) error {
	return r.run("/usr/sbin/tc", args...)
}

func (r *Runner) RunIP(args ...string) error {
	return r.run("/usr/sbin/ip", args...)
}

func (r *Runner) RunModprobe(args ...string) error {
	return r.run("/usr/sbin/modprobe", args...)
}

func (r *Runner) Output(args ...string) (string, error) {
	if r.Verbose {
		fmt.Printf("  > tc %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("/usr/sbin/tc", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *Runner) OutputIP(args ...string) (string, error) {
	if r.Verbose {
		fmt.Printf("  > ip %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("/usr/sbin/ip", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *Runner) run(bin string, args ...string) error {
	cmdStr := fmt.Sprintf("%s %s", bin, strings.Join(args, " "))
	if r.DryRun {
		fmt.Printf("  [dry-run] %s\n", cmdStr)
		return nil
	}
	if r.Verbose {
		fmt.Printf("  > %s\n", cmdStr)
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s (%w)", cmdStr, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// RunIgnoreErr runs a tc command and ignores errors (used in teardown).
func (r *Runner) RunIgnoreErr(args ...string) {
	_ = r.Run(args...)
}

func (r *Runner) RunIPIgnoreErr(args ...string) {
	_ = r.RunIP(args...)
}

// QdiscExists checks if a qdisc with the given handle exists on the device.
func (r *Runner) QdiscExists(dev, handle string) bool {
	out, err := r.Output("qdisc", "show", "dev", dev)
	if err != nil {
		return false
	}
	return strings.Contains(out, handle)
}

// ClassExists checks if a class with the given classid exists on the device.
func (r *Runner) ClassExists(dev, classid string) bool {
	out, err := r.Output("class", "show", "dev", dev)
	if err != nil {
		return false
	}
	return strings.Contains(out, "class htb "+classid+" ")
}

// FilterExistsForIP checks if a u32 filter matching the given IP exists.
func (r *Runner) FilterExistsForIP(dev, ip string) bool {
	out, err := r.Output("filter", "show", "dev", dev)
	if err != nil {
		return false
	}
	// u32 filter output contains the IP in match lines
	return strings.Contains(out, ip)
}
