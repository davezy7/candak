package updater

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.2.1", "0.2.0", 1},
		{"0.2.0", "0.2.1", -1},
		{"0.2.0", "0.2.0", 0},
		{"v0.10.0", "v0.9.9", 1},
		{"1.0.0-beta", "1.0.0", 0},
		{"0.2", "0.2.0", 0},
		{"1.0", "0.9.9", 1},
		{"V0.2.1", "0.2.0", 1},
		{"V0.2.0", "0.2.0", 0},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestPickAssetWindows(t *testing.T) {
	assets := []Asset{
		{Name: "candak-darwin-arm64.dmg"},
		{Name: "candak.exe"},
		{Name: "candak-windows-amd64-installer.exe"},
		{Name: "source.tar.gz"},
	}
	got := pickAsset(assets, "windows", "amd64")
	if got == nil || got.Name != "candak-windows-amd64-installer.exe" {
		t.Fatalf("want installer exe, got %+v", got)
	}
}

func TestPickAssetWindowsPlainFallback(t *testing.T) {
	assets := []Asset{{Name: "candak.exe"}}
	got := pickAsset(assets, "windows", "amd64")
	if got == nil || got.Name != "candak.exe" {
		t.Fatalf("want candak.exe, got %+v", got)
	}
}

func TestPickAssetWindowsNoOsInName(t *testing.T) {
	assets := []Asset{{Name: "candak-setup.exe"}}
	got := pickAsset(assets, "windows", "amd64")
	if got == nil || got.Name != "candak-setup.exe" {
		t.Fatalf("want candak-setup.exe, got %+v", got)
	}
}

func TestPickAssetDarwinArch(t *testing.T) {
	assets := []Asset{
		{Name: "candak-darwin-amd64.dmg"},
		{Name: "candak-darwin-arm64.dmg"},
	}
	got := pickAsset(assets, "darwin", "arm64")
	if got == nil || got.Name != "candak-darwin-arm64.dmg" {
		t.Fatalf("want arm64 dmg, got %+v", got)
	}
}

func TestPickAssetDarwinNoOsInName(t *testing.T) {
	assets := []Asset{
		{Name: "candak-0.2.1-amd64.dmg"},
		{Name: "candak-0.2.1-arm64.dmg"},
	}
	got := pickAsset(assets, "darwin", "arm64")
	if got == nil || got.Name != "candak-0.2.1-arm64.dmg" {
		t.Fatalf("want arm64 dmg without darwin in name, got %+v", got)
	}
}

func TestPickAssetDarwinFallback(t *testing.T) {
	assets := []Asset{{Name: "candak-darwin-amd64.dmg"}}
	got := pickAsset(assets, "darwin", "arm64")
	if got == nil || got.Name != "candak-darwin-amd64.dmg" {
		t.Fatalf("want any darwin dmg fallback, got %+v", got)
	}
}

func TestPickAssetUnsupported(t *testing.T) {
	if got := pickAsset([]Asset{{Name: "candak.exe"}}, "linux", "amd64"); got != nil {
		t.Fatalf("want nil asset on linux, got %+v", got)
	}
}
