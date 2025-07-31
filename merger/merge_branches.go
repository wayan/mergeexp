package merger

import (
	"fmt"

	"log/slog"
)

func (m *Merger) MergeBranches(branches []MergeRef) error {
	for i, b := range branches {
		slog.Info(fmt.Sprintf("Merging %d of %d (%s)", i+1, len(branches), b.Name()))
		if err := m.mergeBranch(b, i, len(branches)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Merger) mergeBranch(b MergeRef, i, n int) error {
	message := fmt.Sprintf("Experimental merge of %s", b.Name())
	if err := m.dir.Command("git", "merge", "--no-ff", "--log", "-m", message, b.Sha()).Run(); err != nil {
		return m.resolveConflict(b, message, 0, i, n)
	}
	return nil
}

func (m *Merger) resolveConflict(b MergeRef, message string, retry, i, n int) error {
	retries := m.ConflictRetries
	if retries == 0 {
		retries = 4
	}
	if retry > retries {
		return fmt.Errorf(fmt.Sprintf("even after %d attempts the working dir is still not clean, aborting", retry))
	}

	hasunmerged := m.dir.Command("git", "diff", "--exit-code", "--quiet", "--diff-filter=U").Run() != nil
	if hasunmerged {
		// are there any unmerged files (--diff-filter=U)
		output, _ := m.dir.Command("git", "diff", "--name-only", "--diff-filter=U").Output()
		// lines := splitLines( output )

		slog.Info(fmt.Sprintf(
			`Conflict in %s, you have unmerged files:
%s
nResolve conflict, commit (or just add the files) and exit the shell (CTRL+D)`,
			b.Name(),
			string(output),
		))

		prompt := m.conflictPrompt(b, retry, i, n)
		if err := m.dir.RunBashWithPrompt(prompt); err != nil {
			return err
		}
		return m.resolveConflict(b, message, retry+1, i, n)
	} else {
		// no conflict - do we have some staged files
		// hascached := me.Command("git", "diff", "--cached", "--exit-code", "--quiet").Run() != nil

		// are we inside merge ?
		cmd := m.dir.Command("git", "rev-parse", "-q", "--verify", "MERGE_HEAD")
		// &>/dev/null
		cmd.Stderr = nil

		if cmd.Run() == nil {
			newMessage := message
			if retry == 0 {
				newMessage = newMessage + " with resolved conflict(s) using rerere"
			}
			return m.dir.Command("git", "commit", "-m", newMessage).Run()
		}
		return nil
	}
}

// conflictPrompt is an informative bash prompt to be displayed on invoked bash
func (m *Merger) conflictPrompt(b MergeRef, retry, i, n int) string {
	name := b.Name()
	if len(name) > 40 {
		name = name[:37] + "..."
	}
	prompt := fmt.Sprintf("Merging %d of %d (%s)", i+1, n, name)
	if retry > 0 {
		prompt += fmt.Sprintf(" (retry %d of %d)", retry, m.ConflictRetries)
	}
	return prompt + "$ "
}
