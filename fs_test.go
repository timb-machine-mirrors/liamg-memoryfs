package memoryfs

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Basics(t *testing.T) {

	memfs := New()

	require.NoError(t, memfs.MkdirAll("files/a/b/c", 0o700))
	require.NoError(t, memfs.WriteFile("test.txt", []byte("hello world"), 0o644))
	require.NoError(t, memfs.WriteFile("files/a/b/c/.secret", []byte("secret file!"), 0o644))
	require.NoError(t, memfs.WriteFile("files/a/b/c/note.txt", []byte(":)"), 0o644))
	require.NoError(t, memfs.WriteFile("files/a/middle.txt", []byte(":("), 0o644))

	t.Run("Open file", func(t *testing.T) {
		f, err := memfs.Open("test.txt")
		require.NoError(t, err)
		data, err := ioutil.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(data))
		require.NoError(t, f.Close())
	})

	t.Run("Open missing file", func(t *testing.T) {
		f, err := memfs.Open("missing.txt")
		require.Error(t, err)
		require.Nil(t, f)
	})

	t.Run("Open directory", func(t *testing.T) {
		f, err := memfs.Open("files")
		require.NoError(t, err)
		require.NotNil(t, f)
		_, err = f.Read([]byte{})
		require.Error(t, err)
	})

	t.Run("Open file in dir", func(t *testing.T) {
		f, err := memfs.Open("files/a/b/c/.secret")
		require.NoError(t, err)
		data, err := ioutil.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, "secret file!", string(data))
		require.NoError(t, f.Close())
	})

	t.Run("Stat file", func(t *testing.T) {
		info, err := memfs.Stat("test.txt")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.Equal(t, fs.FileMode(0o644), info.Mode())
		assert.Equal(t, false, info.IsDir())
		assert.Equal(t, int64(11), info.Size())
	})

	t.Run("Stat file in dir", func(t *testing.T) {
		info, err := memfs.Stat("files/a/b/c/.secret")
		require.NoError(t, err)
		assert.Equal(t, ".secret", info.Name())
		assert.Equal(t, fs.FileMode(0o644), info.Mode())
		assert.Equal(t, false, info.IsDir())
		assert.Equal(t, int64(12), info.Size())
	})

	t.Run("Stat missing file", func(t *testing.T) {
		info, err := memfs.Stat("missing.txt")
		require.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("List directory at root", func(t *testing.T) {
		entries, err := fs.ReadDir(memfs, ".")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assertEntryFound(t, "files", true, entries)
		assertEntryFound(t, "test.txt", false, entries)
	})

	t.Run("List directory with file and dir", func(t *testing.T) {
		entries, err := fs.ReadDir(memfs, "files/a")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assertEntryFound(t, "middle.txt", false, entries)
		assertEntryFound(t, "b", true, entries)
	})

	t.Run("List directory with multiple files", func(t *testing.T) {
		entries, err := fs.ReadDir(memfs, "files/a/b/c")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assertEntryFound(t, ".secret", false, entries)
		assertEntryFound(t, "note.txt", false, entries)
	})

	t.Run("Stat root", func(t *testing.T) {
		info, err := memfs.Stat(".")
		require.NoError(t, err)
		assert.Equal(t, ".", info.Name())
		assert.Equal(t, fs.FileMode(0o700), info.Mode())
		assert.Equal(t, true, info.IsDir())
	})

	t.Run("Walk directory", func(t *testing.T) {
		assertWalkContainsEntries(
			t,
			memfs,
			".",
			[]string{
				"test.txt",
				"files/a/b/c/.secret",
				"files/a/b/c/note.txt",
				"files/a/middle.txt",
			},
			[]string{
				".",
				"files",
				"files/a",
				"files/a/b",
				"files/a/b/c",
			},
		)
	})
}

type entry struct {
	path string
	info fs.DirEntry
}

func assertEntryFound(t *testing.T, expectedName string, expectedDir bool, entries []fs.DirEntry) {
	var count int
	for _, entry := range entries {
		if entry.Name() == expectedName {
			count++
			if count > 1 {
				t.Errorf("entry %s was found more than once", expectedName)
			}
			assert.Equal(t, expectedDir, entry.IsDir(), "%s was not the expected type", expectedName)
		}
	}
	assert.Greater(t, count, 0, "%s was not found", expectedName)
}

func assertWalkContainsEntries(t *testing.T, f fs.FS, dir string, files []string, dirs []string) {
	var entries []entry
	err := fs.WalkDir(f, dir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, entry{
			info: info,
			path: path,
		})
		return nil
	})
	require.NoError(t, err)
	for _, expectedFile := range files {
		var count int
		for _, entry := range entries {
			if entry.path == expectedFile {
				count++
				if entry.info.IsDir() {
					t.Errorf("'%s' should be a file, but is a directory", expectedFile)
				}
			}
		}
		assert.Equal(t, 1, count, fmt.Sprintf("file '%s' should have been found once", expectedFile))
	}
	for _, expectedDir := range dirs {
		var count int
		for _, entry := range entries {
			if entry.path == expectedDir {
				count++
				if !entry.info.IsDir() {
					t.Errorf("'%s' should be a file, but is a directory", expectedDir)
				}
			}
		}
		assert.Equal(t, 1, count, fmt.Sprintf("directory '%s' should have been found once", expectedDir))
	}
}