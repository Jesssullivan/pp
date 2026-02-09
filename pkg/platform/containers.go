package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectContainerRuntimes detects all installed container runtimes on the
// current platform. Returns a slice of detected runtimes (may be empty).
func DetectContainerRuntimes() []ContainerRuntime {
	var runtimes []ContainerRuntime

	if r := plDetectDocker(); r != nil {
		runtimes = append(runtimes, *r)
	}
	if r := plDetectPodman(); r != nil {
		runtimes = append(runtimes, *r)
	}

	// Darwin-only runtimes
	if runtime.GOOS == "darwin" {
		if r := plDetectColima(); r != nil {
			runtimes = append(runtimes, *r)
		}
		if r := plDetectOrbstack(); r != nil {
			runtimes = append(runtimes, *r)
		}
		if r := plDetectLima(); r != nil {
			runtimes = append(runtimes, *r)
		}
	}

	return runtimes
}

// plDetectDocker checks for a Docker installation and whether it is running.
func plDetectDocker() *ContainerRuntime {
	socket := plDockerSocket()
	binary, err := exec.LookPath("docker")
	if err != nil {
		// No docker binary found
		if _, err := os.Stat(socket); err != nil {
			return nil
		}
		// Socket exists but no binary -- report as detected but no version
		return &ContainerRuntime{
			Name:    "docker",
			Running: true,
			Socket:  socket,
		}
	}

	rt := &ContainerRuntime{
		Name:   "docker",
		Socket: socket,
	}

	// Get version
	out, err := exec.Command(binary, "version", "--format", "{{.Server.Version}}").Output()
	if err == nil {
		rt.Version = strings.TrimSpace(string(out))
		rt.Running = true
	} else {
		// Binary exists but daemon may not be running
		rt.Running = false
		// Try client version
		out, err = exec.Command(binary, "version", "--format", "{{.Client.Version}}").Output()
		if err == nil {
			rt.Version = strings.TrimSpace(string(out))
		}
	}

	return rt
}

// plDockerSocket returns the default Docker socket path for the current platform.
func plDockerSocket() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		// Check common Darwin socket locations
		candidates := []string{
			filepath.Join(home, ".colima", "default", "docker.sock"),
			filepath.Join(home, ".orbstack", "run", "docker.sock"),
			"/var/run/docker.sock",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
		return "/var/run/docker.sock"
	}
	return "/var/run/docker.sock"
}

// plDetectPodman checks for a Podman installation and whether it is running.
func plDetectPodman() *ContainerRuntime {
	binary, err := exec.LookPath("podman")
	if err != nil {
		return nil
	}

	rt := &ContainerRuntime{
		Name: "podman",
	}

	// Determine socket path
	uid := os.Getuid()
	if runtime.GOOS == "linux" {
		rt.Socket = filepath.Join("/run", "user", strings.TrimSpace(plItoa(uid)), "podman", "podman.sock")
	} else {
		home, _ := os.UserHomeDir()
		rt.Socket = filepath.Join(home, ".local", "share", "containers", "podman", "machine", "podman.sock")
	}

	// Get version
	out, err := exec.Command(binary, "version", "--format", "{{.Version}}").Output()
	if err == nil {
		rt.Version = strings.TrimSpace(string(out))
		rt.Running = true
	}

	return rt
}

// plDetectColima checks for Colima (Darwin only).
func plDetectColima() *ContainerRuntime {
	binary, err := exec.LookPath("colima")
	if err != nil {
		return nil
	}

	rt := &ContainerRuntime{
		Name: "colima",
	}

	home, _ := os.UserHomeDir()
	rt.Socket = filepath.Join(home, ".colima", "default", "docker.sock")

	// Check if running
	out, err := exec.Command(binary, "status").Output()
	if err == nil {
		status := strings.TrimSpace(string(out))
		rt.Running = strings.Contains(strings.ToLower(status), "running")
	}

	// Get version
	out, err = exec.Command(binary, "version").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "colima version") {
				rt.Version = strings.TrimPrefix(line, "colima version ")
				break
			}
		}
	}

	return rt
}

// plDetectOrbstack checks for OrbStack (Darwin only).
func plDetectOrbstack() *ContainerRuntime {
	binary, err := exec.LookPath("orbctl")
	if err != nil {
		// Also check for orbstack binary
		binary, err = exec.LookPath("orbstack")
		if err != nil {
			return nil
		}
	}

	rt := &ContainerRuntime{
		Name: "orbstack",
	}

	home, _ := os.UserHomeDir()
	rt.Socket = filepath.Join(home, ".orbstack", "run", "docker.sock")

	// Check if running by looking for the socket
	if _, err := os.Stat(rt.Socket); err == nil {
		rt.Running = true
	}

	// Get version
	out, err := exec.Command(binary, "version").Output()
	if err == nil {
		rt.Version = strings.TrimSpace(string(out))
	}

	return rt
}

// plDetectLima checks for Lima VM instances (Darwin only).
func plDetectLima() *ContainerRuntime {
	binary, err := exec.LookPath("limactl")
	if err != nil {
		return nil
	}

	rt := &ContainerRuntime{
		Name: "lima",
	}

	home, _ := os.UserHomeDir()
	rt.Socket = filepath.Join(home, ".lima", "default", "sock", "dockerd.sock")

	// Check if any instance is running
	out, err := exec.Command(binary, "list", "--format", "{{.Status}}").Output()
	if err == nil {
		statuses := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, s := range statuses {
			if strings.TrimSpace(s) == "Running" {
				rt.Running = true
				break
			}
		}
	}

	// Get version
	out, err = exec.Command(binary, "--version").Output()
	if err == nil {
		rt.Version = strings.TrimSpace(string(out))
	}

	return rt
}

// plItoa converts an int to a string without importing strconv in this file.
func plItoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + plItoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
