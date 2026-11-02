// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apps

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, text string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}

// TestXDG tests the freedesktop implementation, with XDG directories of the
// test, which runs on all platforms.
func TestXDG(t *testing.T) {
	tmp := t.TempDir()
	data, sysData := filepath.Join(tmp, "data"), filepath.Join(tmp, "sysdata")
	config := filepath.Join(tmp, "config")
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_DATA_DIRS", sysData)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_CONFIG_DIRS", filepath.Join(tmp, "sysconfig"))
	entry := func(name, exec, mime string, more ...string) string {
		return "[Desktop Entry]\nType=Application\nName=" + name + "\nExec=" + exec + "\nMimeType=" + mime + "\n" + strings.Join(more, "\n") + "\n[Desktop Action new]\nName=Not this\nHidden=true\n"
	}
	writeFile(t, filepath.Join(sysData, "applications", "editor.desktop"), entry("Editor", "editor %F", "text/plain;text/x-go;"))
	writeFile(t, filepath.Join(sysData, "applications", "kde", "viewer.desktop"), entry("Viewer", "viewer --file=%f", "text/plain;"))
	writeFile(t, filepath.Join(sysData, "applications", "gone.desktop"), entry("Gone", "gone", "text/plain", "Hidden=true"))
	writeFile(t, filepath.Join(sysData, "applications", "removed.desktop"), entry("Removed", "removed", "text/plain"))
	writeFile(t, filepath.Join(sysData, "applications", "link.desktop"), "[Desktop Entry]\nType=Link\nName=Link\nMimeType=text/plain\n")
	writeFile(t, filepath.Join(data, "applications", "notes.desktop"), entry("Notes", "notes", "text/markdown"))
	writeFile(t, filepath.Join(sysData, "applications", "mimeapps.list"),
		"[Default Applications]\ntext/plain=missing.desktop;kde-viewer.desktop;\n[Added Associations]\ntext/plain=notes.desktop;\n[Removed Associations]\ntext/plain=removed.desktop;\n")

	names := func(as []App) []string {
		var nms []string
		for _, a := range as {
			nms = append(nms, a.Name)
		}
		return nms
	}
	as := xdgApps("text/plain")
	assert.Equal(t, []string{"Viewer", "Notes", "Editor"}, names(as), "default, added, then by name")
	assert.Equal(t, App{Name: "Viewer", ID: "kde-viewer.desktop", Default: true}, as[0])
	assert.False(t, as[1].Default)
	assert.Equal(t, []string{"Editor"}, names(xdgApps("text/x-go")))
	assert.Empty(t, xdgApps("image/png"))

	// setting the default keeps the other contents of the user's file
	writeFile(t, filepath.Join(config, "mimeapps.list"), "[Added Associations]\nimage/png=editor.desktop;\n\n[Default Applications]\ntext/x-go=editor.desktop;\n")
	require.NoError(t, setXDGDefault("text/plain", "editor.desktop"))
	as = xdgApps("text/plain")
	assert.Equal(t, "Editor", as[0].Name)
	assert.True(t, as[0].Default)
	require.NoError(t, setXDGDefault("text/plain", "notes.desktop")) // replaced
	assert.Equal(t, "Notes", xdgApps("text/plain")[0].Name)
	b, err := os.ReadFile(filepath.Join(config, "mimeapps.list"))
	require.NoError(t, err)
	assert.Equal(t, "[Added Associations]\nimage/png=editor.desktop;\n\n[Default Applications]\ntext/x-go=editor.desktop;\ntext/plain=notes.desktop;\n", string(b))
	assert.Equal(t, []string{"Editor"}, names(xdgApps("image/png")))

	// a new file
	require.NoError(t, os.Remove(filepath.Join(config, "mimeapps.list")))
	require.NoError(t, setXDGDefault("text/plain", "editor.desktop"))
	b, err = os.ReadFile(filepath.Join(config, "mimeapps.list"))
	require.NoError(t, err)
	assert.Equal(t, "[Default Applications]\ntext/plain=editor.desktop;\n", string(b))
}

func TestExecArgs(t *testing.T) {
	file := "/home/me/my notes.txt"
	for _, tc := range []struct {
		exec string
		args []string
	}{
		{"editor %F", []string{"editor", file}},
		{"viewer --file=%f --new-window", []string{"viewer", "--file=" + file, "--new-window"}},
		{"browser %U", []string{"browser", "file:///home/me/my%20notes.txt"}},
		{`"/opt/my app/run" %i %c %k %f`, []string{"/opt/my app/run", file}},
		{`sh -c "echo \"hi\" 100%%"`, []string{"sh", "-c", `echo "hi" 100%`, file}},
		{"plain", []string{"plain", file}},
	} {
		args, err := execArgs(tc.exec, file)
		require.NoError(t, err, tc.exec)
		assert.Equal(t, tc.args, args, tc.exec)
	}
	_, err := execArgs("  ", file)
	assert.Error(t, err)
}

// TestFor tests the apps for a text file on macOS, which has TextEdit.
// The apps are not opened, and the defaults are not changed.
func TestFor(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("uses the apps of macOS")
	}
	file := filepath.Join(t.TempDir(), "a.txt")
	writeFile(t, file, "text\n")
	as, err := For(file)
	require.NoError(t, err)
	require.NotEmpty(t, as)
	assert.True(t, as[0].Default)
	for _, a := range as[1:] {
		assert.False(t, a.Default, a.Name)
	}
	assert.True(t, slices.ContainsFunc(as, func(a App) bool { return a.Name == "TextEdit" }), "TextEdit opens text files")
	for _, a := range as {
		assert.NotEmpty(t, a.Name)
		assert.DirExists(t, a.ID, "the app bundle")
	}
	def, ok, err := Default(file)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, as[0], def)
	t.Logf("apps for a .txt file: %v", as)
}
