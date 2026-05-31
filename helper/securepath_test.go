package helper

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveUnderBase(t *testing.T) {
	base := t.TempDir()
	pics := filepath.Join(base, "pics")
	secret := filepath.Join(base, "secret")
	require.NoError(t, os.MkdirAll(pics, 0o755))
	require.NoError(t, os.MkdirAll(secret, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pics, "allowed.jpg"), []byte("ok"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(secret, "leaked.jpg"), []byte("secret"), 0o644))

	t.Run("allows normal path", func(t *testing.T) {
		abs, rel, err := ResolveUnderBase(pics, "/allowed.jpg")
		require.NoError(t, err)
		assert.Equal(t, "allowed.jpg", rel)
		assert.Equal(t, filepath.Join(pics, "allowed.jpg"), abs)
	})

	t.Run("blocks double encoded traversal", func(t *testing.T) {
		_, _, err := ResolveUnderBase(pics, "/%252E%252E%252Fsecret/leaked.jpg")
		assert.ErrorIs(t, err, ErrPathTraversal)
	})

	t.Run("blocks single encoded traversal", func(t *testing.T) {
		_, _, err := ResolveUnderBase(pics, "/%2E%2E%2Fsecret/leaked.jpg")
		assert.ErrorIs(t, err, ErrPathTraversal)
	})

	t.Run("blocks literal traversal", func(t *testing.T) {
		_, _, err := ResolveUnderBase(pics, "/../secret/leaked.jpg")
		assert.ErrorIs(t, err, ErrPathTraversal)
	})
}

func TestFullyUnescapePath(t *testing.T) {
	out, err := FullyUnescapePath("/%252E%252E%252Fsecret")
	require.NoError(t, err)
	assert.Equal(t, "/../secret", out)
}

func TestRelPathUnderBase(t *testing.T) {
	base := t.TempDir()
	pics := filepath.Join(base, "pics")
	require.NoError(t, os.MkdirAll(pics, 0o755))
	file := filepath.Join(pics, "allowed.jpg")
	require.NoError(t, os.WriteFile(file, []byte("ok"), 0o644))

	rel, err := RelPathUnderBase(pics, file)
	require.NoError(t, err)
	assert.Equal(t, "allowed.jpg", rel)
}

func TestResolveUnderBaseBlocksSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	pics := filepath.Join(base, "pics")
	outsideDir := filepath.Join(base, "outside")
	require.NoError(t, os.MkdirAll(pics, 0o755))
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.jpg"), []byte("secret"), 0o644))

	linkPath := filepath.Join(pics, "escape")
	err := os.Symlink(outsideDir, linkPath)
	if err != nil {
		if runtime.GOOS == "windows" || errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink not supported in current environment: %v", err)
		}
		require.NoError(t, err)
	}

	_, _, resolveErr := ResolveUnderBase(pics, "/escape/secret.jpg")
	assert.ErrorIs(t, resolveErr, ErrPathTraversal)
}
