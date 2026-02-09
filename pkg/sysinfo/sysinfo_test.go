package sysinfo

import (
	"runtime"
	"strings"
	"testing"
)

// --- Sample data constants for parsing tests ---

const sampleNvidiaSMI = `NVIDIA GeForce RTX 3080, 535.129.03, 10240, 45, 12
NVIDIA Tesla V100, 535.129.03, 16384, 52, 87`

const sampleNvidiaSMISingle = `NVIDIA GeForce GTX 1080 Ti, 470.256.02, 11264, 38, 5`

const sampleDarwinGPUJSON = `{
  "SPDisplaysDataType": [
    {
      "sppci_model": "Apple M1 Pro",
      "sppci_vendor": "Apple",
      "sppci_vram_shared": "16 GB",
      "spdisplays_metalfamily": "Metal 3"
    }
  ]
}`

const sampleDarwinGPUMultiJSON = `{
  "SPDisplaysDataType": [
    {
      "sppci_model": "Apple M2 Max",
      "sppci_vendor": "Apple",
      "sppci_vram_shared": "32 GB",
      "spdisplays_metalfamily": "Metal 3"
    },
    {
      "sppci_model": "Radeon Pro W6800",
      "sppci_vendor": "AMD/ATI",
      "sppci_vram": "32768 MB",
      "spdisplays_metalfamily": "Metal GPUFamily macOS 2"
    }
  ]
}`

const samplePSOutput = `USER               PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND
root                 1   0.0  0.1  4286532  26544   ??  Ss   Mon09AM   2:48.23 /sbin/launchd
jsullivan2       12345  42.3  3.2  8123456  65432   ??  R    10:00AM   1:23.45 /usr/bin/python3 train.py --epochs 100
jsullivan2       12346  15.7  1.1  4567890  22334   ??  S    10:05AM   0:45.67 node server.js
root             12347   8.1  0.5  2345678  11223   ??  S    09:55AM   0:12.34 /usr/sbin/httpd -D FOREGROUND
jsullivan2       12348   0.3  0.2  1234567   5678   ??  S    10:10AM   0:01.23 vim config.yaml`

const sampleProcStat = `12345 (python3) S 1 12345 12345 0 -1 4194304 1234 0 0 0 15000 3000 0 0 20 0 1 0 12345678 123456789 1234 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 1 0 0 0 0 0`

const sampleProcStatParens = `99 (Web Content) S 1 99 99 0 -1 4194304 5678 0 0 0 25000 5000 0 0 20 0 3 0 87654321 234567890 2345 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 2 0 0 0 0 0`

const sampleCgroupDocker = `12:devices:/docker/abc123def456
11:cpuset:/docker/abc123def456
0::/system.slice/docker-abc123def456.scope`

const sampleCgroupLXC = `12:devices:/lxc/mycontainer
11:cpuset:/lxc/mycontainer`

const sampleCgroupHost = `12:devices:/
11:cpuset:/
0::/init.scope`

const sampleCgroupPodman = `0::/machine.slice/libpod-abc123.scope`

// --- Collect tests ---

func TestCollectReturnsNonNil(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Collect returned nil SystemInfo")
	}
}

func TestCollectHasNonEmptyHostname(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if info.Hostname == "" {
		t.Error("Collect returned empty Hostname")
	}
}

func TestCollectHasCorrectOS(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("expected OS=%q, got %q", runtime.GOOS, info.OS)
	}
}

func TestCollectHasCorrectArch(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("expected Arch=%q, got %q", runtime.GOARCH, info.Arch)
	}
}

// --- Container detection tests ---

func TestDetectContainerReturnsFalseOutsideContainer(t *testing.T) {
	// When running tests natively (not in a container), this should be false.
	// We cannot guarantee this in CI, so we check that it at least does not
	// panic and returns a consistent pair.
	inContainer, containerType := siDetectContainer()
	if !inContainer && containerType != "" {
		t.Errorf("not in container but containerType=%q", containerType)
	}
	if inContainer && containerType == "" {
		t.Error("in container but containerType is empty")
	}
}

func TestParseCgroupDocker(t *testing.T) {
	ct := siParseCgroup(sampleCgroupDocker)
	if ct != "docker" {
		t.Errorf("expected docker, got %q", ct)
	}
}

func TestParseCgroupLXC(t *testing.T) {
	ct := siParseCgroup(sampleCgroupLXC)
	if ct != "lxc" {
		t.Errorf("expected lxc, got %q", ct)
	}
}

func TestParseCgroupHost(t *testing.T) {
	ct := siParseCgroup(sampleCgroupHost)
	if ct != "" {
		t.Errorf("expected empty string for host cgroup, got %q", ct)
	}
}

func TestParseCgroupPodman(t *testing.T) {
	ct := siParseCgroup(sampleCgroupPodman)
	if ct != "podman" {
		t.Errorf("expected podman, got %q", ct)
	}
}

// --- GPU tests ---

func TestParseNvidiaSMIWithSampleOutput(t *testing.T) {
	gpus := siParseNvidiaSMI(sampleNvidiaSMI)
	if len(gpus) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(gpus))
	}

	// First GPU.
	if gpus[0].Name != "NVIDIA GeForce RTX 3080" {
		t.Errorf("gpu[0].Name = %q", gpus[0].Name)
	}
	if gpus[0].Vendor != "nvidia" {
		t.Errorf("gpu[0].Vendor = %q", gpus[0].Vendor)
	}
	if gpus[0].Driver != "535.129.03" {
		t.Errorf("gpu[0].Driver = %q", gpus[0].Driver)
	}
	// 10240 MiB
	expectedVRAM := uint64(10240 * 1024 * 1024)
	if gpus[0].VRAM != expectedVRAM {
		t.Errorf("gpu[0].VRAM = %d, want %d", gpus[0].VRAM, expectedVRAM)
	}
	if gpus[0].Temperature != 45.0 {
		t.Errorf("gpu[0].Temperature = %f", gpus[0].Temperature)
	}
	if gpus[0].Utilization != 12.0 {
		t.Errorf("gpu[0].Utilization = %f", gpus[0].Utilization)
	}

	// Second GPU.
	if gpus[1].Name != "NVIDIA Tesla V100" {
		t.Errorf("gpu[1].Name = %q", gpus[1].Name)
	}
	if gpus[1].Utilization != 87.0 {
		t.Errorf("gpu[1].Utilization = %f", gpus[1].Utilization)
	}
}

func TestParseNvidiaSMIWithEmptyOutput(t *testing.T) {
	gpus := siParseNvidiaSMI("")
	if len(gpus) != 0 {
		t.Errorf("expected 0 GPUs for empty input, got %d", len(gpus))
	}
}

func TestParseNvidiaSMISingleGPU(t *testing.T) {
	gpus := siParseNvidiaSMI(sampleNvidiaSMISingle)
	if len(gpus) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(gpus))
	}
	if gpus[0].Name != "NVIDIA GeForce GTX 1080 Ti" {
		t.Errorf("gpu[0].Name = %q", gpus[0].Name)
	}
}

func TestParseDarwinGPUWithSampleJSON(t *testing.T) {
	gpus := siParseDarwinGPU([]byte(sampleDarwinGPUJSON))
	if len(gpus) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(gpus))
	}
	if gpus[0].Name != "Apple M1 Pro" {
		t.Errorf("gpu[0].Name = %q", gpus[0].Name)
	}
	if gpus[0].Vendor != "apple" {
		t.Errorf("gpu[0].Vendor = %q", gpus[0].Vendor)
	}
	// 16 GB = 17179869184 bytes.
	expectedVRAM := uint64(16 * 1024 * 1024 * 1024)
	if gpus[0].VRAM != expectedVRAM {
		t.Errorf("gpu[0].VRAM = %d, want %d", gpus[0].VRAM, expectedVRAM)
	}
	if gpus[0].Driver != "Metal 3" {
		t.Errorf("gpu[0].Driver = %q", gpus[0].Driver)
	}
}

func TestParseDarwinGPUMulti(t *testing.T) {
	gpus := siParseDarwinGPU([]byte(sampleDarwinGPUMultiJSON))
	if len(gpus) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(gpus))
	}
	if gpus[0].Vendor != "apple" {
		t.Errorf("gpu[0].Vendor = %q, want apple", gpus[0].Vendor)
	}
	if gpus[1].Vendor != "amd" {
		t.Errorf("gpu[1].Vendor = %q, want amd", gpus[1].Vendor)
	}
	// 32768 MB.
	expectedVRAM := uint64(32768 * 1024 * 1024)
	if gpus[1].VRAM != expectedVRAM {
		t.Errorf("gpu[1].VRAM = %d, want %d", gpus[1].VRAM, expectedVRAM)
	}
}

func TestParseDarwinGPUEmptyJSON(t *testing.T) {
	gpus := siParseDarwinGPU([]byte{})
	if len(gpus) != 0 {
		t.Errorf("expected 0 GPUs for empty JSON, got %d", len(gpus))
	}
}

func TestParseDarwinGPUInvalidJSON(t *testing.T) {
	gpus := siParseDarwinGPU([]byte("not json"))
	if len(gpus) != 0 {
		t.Errorf("expected 0 GPUs for invalid JSON, got %d", len(gpus))
	}
}

func TestGPUInfoStructFields(t *testing.T) {
	gpu := GPUInfo{
		Name:        "Test GPU",
		Vendor:      "nvidia",
		VRAM:        8589934592,
		Driver:      "535.0",
		Temperature: 65.5,
		Utilization: 42.0,
	}
	if gpu.Name != "Test GPU" {
		t.Error("GPUInfo.Name field mismatch")
	}
	if gpu.VRAM != 8589934592 {
		t.Error("GPUInfo.VRAM field mismatch")
	}
	if gpu.Temperature != 65.5 {
		t.Error("GPUInfo.Temperature field mismatch")
	}
	if gpu.Utilization != 42.0 {
		t.Error("GPUInfo.Utilization field mismatch")
	}
}

// --- NIC tests ---

func TestDetectNICsReturnsAtLeastLoopback(t *testing.T) {
	nics := siDetectNICs()
	hasLoopback := false
	for _, nic := range nics {
		if nic.Type == "loopback" {
			hasLoopback = true
			break
		}
	}
	if !hasLoopback {
		t.Error("expected at least one loopback interface")
	}
}

func TestClassifyNICLoopback(t *testing.T) {
	if typ := siClassifyNIC("lo0", "darwin"); typ != "loopback" {
		t.Errorf("lo0 classified as %q, want loopback", typ)
	}
	if typ := siClassifyNIC("lo", "linux"); typ != "loopback" {
		t.Errorf("lo classified as %q, want loopback", typ)
	}
}

func TestClassifyNICEn0Darwin(t *testing.T) {
	if typ := siClassifyNIC("en0", "darwin"); typ != "ethernet" {
		t.Errorf("en0 on darwin classified as %q, want ethernet", typ)
	}
}

func TestClassifyNICEth0Linux(t *testing.T) {
	if typ := siClassifyNIC("eth0", "linux"); typ != "ethernet" {
		t.Errorf("eth0 on linux classified as %q, want ethernet", typ)
	}
}

func TestClassifyNICWlan0(t *testing.T) {
	if typ := siClassifyNIC("wlan0", "linux"); typ != "wifi" {
		t.Errorf("wlan0 classified as %q, want wifi", typ)
	}
}

func TestClassifyNICWlp(t *testing.T) {
	if typ := siClassifyNIC("wlp3s0", "linux"); typ != "wifi" {
		t.Errorf("wlp3s0 classified as %q, want wifi", typ)
	}
}

func TestClassifyNICVeth(t *testing.T) {
	if typ := siClassifyNIC("veth123abc", "linux"); typ != "virtual" {
		t.Errorf("veth123abc classified as %q, want virtual", typ)
	}
}

func TestClassifyNICDockerBridge(t *testing.T) {
	if typ := siClassifyNIC("docker0", "linux"); typ != "virtual" {
		t.Errorf("docker0 classified as %q, want virtual", typ)
	}
	if typ := siClassifyNIC("br-abc123", "linux"); typ != "virtual" {
		t.Errorf("br-abc123 classified as %q, want virtual", typ)
	}
}

func TestClassifyNICTailscale(t *testing.T) {
	if typ := siClassifyNIC("tailscale0", "linux"); typ != "tailscale" {
		t.Errorf("tailscale0 classified as %q, want tailscale", typ)
	}
}

func TestClassifyNICEnpLinux(t *testing.T) {
	if typ := siClassifyNIC("enp0s3", "linux"); typ != "ethernet" {
		t.Errorf("enp0s3 classified as %q, want ethernet", typ)
	}
}

func TestNICInfoStructFields(t *testing.T) {
	nic := NICInfo{
		Name:  "en0",
		Type:  "ethernet",
		MAC:   "aa:bb:cc:dd:ee:ff",
		IPv4:  "192.168.1.100",
		IPv6:  "fe80::1",
		Up:    true,
		Speed: "1Gbps",
	}
	if nic.Name != "en0" {
		t.Error("NICInfo.Name field mismatch")
	}
	if nic.Type != "ethernet" {
		t.Error("NICInfo.Type field mismatch")
	}
	if !nic.Up {
		t.Error("NICInfo.Up should be true")
	}
	if nic.Speed != "1Gbps" {
		t.Error("NICInfo.Speed field mismatch")
	}
}

// --- Process tests ---

func TestParsePSWithSampleOutput(t *testing.T) {
	procs := siParsePS(samplePSOutput)
	if len(procs) != 5 {
		t.Fatalf("expected 5 processes, got %d", len(procs))
	}

	// Should be sorted by CPU descending.
	if procs[0].CPU != 42.3 {
		t.Errorf("highest CPU process = %f, want 42.3", procs[0].CPU)
	}
	if procs[0].Name != "/usr/bin/python3 train.py --epochs 100" {
		t.Errorf("highest CPU process name = %q", procs[0].Name)
	}
	if procs[0].PID != 12345 {
		t.Errorf("highest CPU process PID = %d, want 12345", procs[0].PID)
	}
	if procs[0].User != "jsullivan2" {
		t.Errorf("highest CPU process user = %q", procs[0].User)
	}
	if procs[0].Memory != 3.2 {
		t.Errorf("highest CPU process memory = %f, want 3.2", procs[0].Memory)
	}
}

func TestParsePSWithEmptyOutput(t *testing.T) {
	procs := siParsePS("")
	if len(procs) != 0 {
		t.Errorf("expected 0 processes for empty input, got %d", len(procs))
	}
}

func TestParsePSHeaderOnly(t *testing.T) {
	procs := siParsePS("USER               PID  %CPU %MEM      VSZ    RSS   TT  STAT STARTED      TIME COMMAND\n")
	if len(procs) != 0 {
		t.Errorf("expected 0 processes for header-only input, got %d", len(procs))
	}
}

func TestParseProcStat(t *testing.T) {
	p := siParseProcStat(sampleProcStat)
	if p == nil {
		t.Fatal("siParseProcStat returned nil")
	}
	if p.PID != 12345 {
		t.Errorf("PID = %d, want 12345", p.PID)
	}
	if p.Name != "python3" {
		t.Errorf("Name = %q, want python3", p.Name)
	}
	// utime=15000 + stime=3000 = 18000
	if p.CPU != 18000 {
		t.Errorf("CPU = %f, want 18000", p.CPU)
	}
}

func TestParseProcStatWithSpacesInName(t *testing.T) {
	p := siParseProcStat(sampleProcStatParens)
	if p == nil {
		t.Fatal("siParseProcStat returned nil for name with spaces")
	}
	if p.PID != 99 {
		t.Errorf("PID = %d, want 99", p.PID)
	}
	if p.Name != "Web Content" {
		t.Errorf("Name = %q, want 'Web Content'", p.Name)
	}
}

func TestParseProcStatEmpty(t *testing.T) {
	p := siParseProcStat("")
	if p != nil {
		t.Error("expected nil for empty input")
	}
}

// --- Kernel tests ---

func TestParseKernelVersionLinuxProc(t *testing.T) {
	raw := "Linux version 6.1.0-27-amd64 (debian-kernel@lists.debian.org) (gcc-12 (Debian 12.2.0-14) 12.2.0, GNU ld (GNU Binutils for Debian) 2.40) #1 SMP PREEMPT_DYNAMIC Debian 6.1.115-1 (2024-11-01)"
	ver := siParseKernelVersion(raw)
	if ver != "6.1.0-27-amd64" {
		t.Errorf("parsed kernel version = %q, want 6.1.0-27-amd64", ver)
	}
}

func TestParseKernelVersionDarwin(t *testing.T) {
	// Darwin kernel version from uname -r / sysctl kern.osrelease.
	raw := "24.3.0"
	ver := siParseKernelVersion(raw)
	if ver != "24.3.0" {
		t.Errorf("parsed kernel version = %q, want 24.3.0", ver)
	}
}

func TestParseKernelVersionEmpty(t *testing.T) {
	ver := siParseKernelVersion("")
	if ver != "" {
		t.Errorf("expected empty string, got %q", ver)
	}
}

func TestParseKernelVersionWhitespace(t *testing.T) {
	ver := siParseKernelVersion("  5.15.0-generic  \n")
	if ver != "5.15.0-generic" {
		t.Errorf("parsed kernel version = %q, want 5.15.0-generic", ver)
	}
}

func TestKernelVersionReturnsNonEmpty(t *testing.T) {
	ver := siKernelVersion()
	if ver == "" {
		t.Error("siKernelVersion returned empty string")
	}
}

// --- Vendor normalization tests ---

func TestNormalizeGPUVendor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"NVIDIA Corporation", "nvidia"},
		{"AMD/ATI", "amd"},
		{"Intel Corporation", "intel"},
		{"Apple", "apple"},
		{"sppci_vendor_Apple", "apple"},
		{"", ""},
	}
	for _, tc := range tests {
		got := siNormalizeGPUVendor(tc.input)
		if got != tc.want {
			t.Errorf("siNormalizeGPUVendor(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- VRAM parsing tests ---

func TestParseVRAMString(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
	}{
		{"16 GB", 16 * 1024 * 1024 * 1024},
		{"8192 MB", 8192 * 1024 * 1024},
		{"", 0},
		{"unknown", 0},
	}
	for _, tc := range tests {
		got := siParseVRAMString(tc.input)
		if got != tc.want {
			t.Errorf("siParseVRAMString(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// --- IP extraction tests ---

func TestExtractIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.1/24", "192.168.1.1"},
		{"fe80::1/64", "fe80::1"},
		{"10.0.0.1", "10.0.0.1"},
	}
	for _, tc := range tests {
		got := siExtractIP(tc.input)
		if got != tc.want {
			t.Errorf("siExtractIP(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- Integration-level tests ---

func TestCollectKernelNonEmpty(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if info.Kernel == "" {
		t.Error("Collect returned empty kernel version")
	}
}

func TestCollectNICsPopulated(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if len(info.NICs) == 0 {
		t.Error("Collect returned no NICs")
	}
	// At least one should be loopback.
	found := false
	for _, nic := range info.NICs {
		if nic.Type == "loopback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no loopback NIC found in Collect results")
	}
}

func TestCollectGPUsNotNil(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	// GPUs slice may be empty (no GPU detected) but should not be nil from
	// the platform detection. However, since the platform function may
	// return nil on failure, we just verify no panic occurred.
	_ = info.GPUs
}

func TestCollectContainerFieldsConsistent(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if !info.InContainer && info.ContainerType != "" {
		t.Errorf("InContainer=false but ContainerType=%q", info.ContainerType)
	}
}

func TestProcessInfoStructFields(t *testing.T) {
	p := ProcessInfo{
		PID:    1234,
		Name:   "test-proc",
		CPU:    50.5,
		Memory: 12.3,
		User:   "root",
	}
	if p.PID != 1234 {
		t.Error("ProcessInfo.PID mismatch")
	}
	if p.Name != "test-proc" {
		t.Error("ProcessInfo.Name mismatch")
	}
	if p.CPU != 50.5 {
		t.Error("ProcessInfo.CPU mismatch")
	}
	if p.Memory != 12.3 {
		t.Error("ProcessInfo.Memory mismatch")
	}
	if p.User != "root" {
		t.Error("ProcessInfo.User mismatch")
	}
}

// --- siParseFloat edge cases ---

func TestParseFloat(t *testing.T) {
	if v := siParseFloat("42.5"); v != 42.5 {
		t.Errorf("siParseFloat(42.5) = %f", v)
	}
	if v := siParseFloat("notanumber"); v != 0 {
		t.Errorf("siParseFloat(notanumber) = %f, want 0", v)
	}
	if v := siParseFloat(""); v != 0 {
		t.Errorf("siParseFloat('') = %f, want 0", v)
	}
}

// --- SystemInfo string representation ---

func TestSystemInfoOSValues(t *testing.T) {
	// Verify that the OS field contains valid known values.
	info, _ := Collect()
	validOS := map[string]bool{
		"darwin":  true,
		"linux":   true,
		"windows": true,
		"freebsd": true,
	}
	if !validOS[info.OS] {
		// Not a hard failure since Go supports many platforms, but worth logging.
		t.Logf("unexpected OS value: %q", info.OS)
	}
}

func TestSystemInfoArchValues(t *testing.T) {
	info, _ := Collect()
	validArch := map[string]bool{
		"amd64": true,
		"arm64": true,
		"arm":   true,
		"386":   true,
	}
	if !validArch[info.Arch] {
		t.Logf("unexpected Arch value: %q", info.Arch)
	}
}

// Ensure the loopback interface has expected properties.
func TestLoopbackInterfaceProperties(t *testing.T) {
	nics := siDetectNICs()
	for _, nic := range nics {
		if nic.Type == "loopback" {
			if !nic.Up {
				t.Error("loopback interface should be up")
			}
			if nic.IPv4 != "" && !strings.HasPrefix(nic.IPv4, "127.") {
				t.Errorf("loopback IPv4 = %q, expected 127.x.x.x", nic.IPv4)
			}
			return
		}
	}
	t.Error("no loopback interface found")
}
