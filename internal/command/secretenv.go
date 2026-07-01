package command

import "github.com/svyatov/oz/internal/config"

// SecretEnv returns NAME=value entries for password options that declare a
// secret_env channel and have a non-empty value. These are delivered through
// the child process environment (via RunWithEnv), never through argv, so the
// secret stays off the process list (see docs for platform scoping).
func SecretEnv(w *config.Wizard, values config.Values) []string {
	var env []string
	for _, opt := range w.Options {
		if opt.Type != config.OptionPassword || opt.SecretEnv == "" {
			continue
		}
		val, ok := values[opt.Name]
		if !ok {
			continue
		}
		if s := val.Scalar(); s != "" {
			env = append(env, opt.SecretEnv+"="+s)
		}
	}
	return env
}
