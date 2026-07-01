package command

import "testing"

// TestRunWithEnv verifies the child process sees env vars passed via RunWithEnv,
// and does not see them when none are passed. The child asserts via its exit code.
func TestRunWithEnv(t *testing.T) {
	check := []string{"sh", "-c", `[ "$OZ_TEST_SECRET" = "abc123" ]`}

	t.Run("var_visible_to_child", func(t *testing.T) {
		if err := RunWithEnv(check, []string{"OZ_TEST_SECRET=abc123"}); err != nil {
			t.Errorf("child did not see injected env var: %v", err)
		}
	})

	t.Run("absent_when_no_env", func(t *testing.T) {
		if err := RunWithEnv(check, nil); err == nil {
			t.Error("expected child to fail with env var unset, got nil")
		}
	})
}
