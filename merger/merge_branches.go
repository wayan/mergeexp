package merger

import (
	"fmt"

	"log/slog"
)

func (m *Merger) MergeBranches(branches []MergeRef) error {
	cnt := len(branches)
	i := 0
	for _, b := range branches {
		i = i + 1
		slog.Info(fmt.Sprintf("Merging %d of %d (%s)", i, cnt, b.Name()))
		if err := m.MergeBranch(b); err != nil {
			return err
		}
	}
	return nil
}

func (m *Merger) MergeBranch(b MergeRef) error {
	message := fmt.Sprintf("Experimental merge of %s", b.Name())
	err := m.dir.Command("git", "merge", "--no-ff", "--log", "-m", message, b.Sha()).Run()
	if err != nil {
		return m.ResolveConflict(b, message)
	}
	return nil
}

func (m *Merger) ResolveConflict(b MergeRef, message string) error {
	return m.resolveConflict(b, message, 0)
}

func (m *Merger) resolveConflict(b MergeRef, message string, retry int) error {
	maxRetries := m.ConflictRetries
	if maxRetries == 0 {
		maxRetries = 4
	}
	if retry > maxRetries {
		return fmt.Errorf(fmt.Sprintf("even after %d attempts the working dir is still not clean, aborting", retry))
	}

	hasunmerged := m.dir.Command("git", "diff", "--exit-code", "--quiet", "--diff-filter=U").Run() != nil
	if hasunmerged {
		// are there any unmerged files (--diff-filter=U)
		output, _ := m.dir.Command("git", "diff", "--name-only", "--diff-filter=U").Output()
		// lines := splitLines( output )

		slog.Info(fmt.Sprintf(
			"Conflict in %s, you have unmerged files:\n%s\nResolve conflict, commit (or just add the files) and exit the shell",
			b.Name,
			string(output),
		))

		retryStr := ""
		if retry > 0 {
			retryStr = fmt.Sprintf(" (retry %d of %d)", retry, maxRetries)
		}

		prompt := fmt.Sprintf("conflict resolution of '%s'%s $ ",
			b.Name,
			retryStr,
		)
		err := m.dir.RunBashWithPrompt(prompt)
		if err != nil {
			return err
		}
		return m.resolveConflict(b, message, retry+1)
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
