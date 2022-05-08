package mergeexp

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type logger interface{ Info(string) }

type MergeExp struct {
	Dir        string
	Logger     logger
	HttpClient *http.Client

	BitBucketUsername      string
	BitBucketPassword      string
	BitBucketApiRoot       string
	BitBucketCloneBase     string
	BitBucketDeploymentKey string

	GitlabCloneBase        string
	ConflictRetries int
}

func (me *MergeExp) Init() *MergeExp {
	if me.HttpClient == nil {
		me.HttpClient = &http.Client{}
	}
	if me.Logger == nil {
		me.Logger = &PrintLogger{}
	}
	return me
}

type PrintLogger struct{}

func (l *PrintLogger) Info(s string) {
	fmt.Println(s)
}

func (me *MergeExp) Command(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = me.Dir
	cmd.Stderr = os.Stderr
	return cmd
}

func (me *MergeExp) GitInit() error {
	err := me.Command("git", "status").Run()
	if err != nil {
		err = me.Command("git", "init").Run()
	}
	return err
}

func (g *MergeExp) StartBranch(branch string, target *Branch) error {
	err := g.Command("git", "branch", "-f", branch, target.Name).Run()
	if err == nil {
		err = g.Command("git", "checkout", branch).Run()
	}
	return err
}

func (me *MergeExp) MergeBranches(branches []Branch) error {
	cnt := len(branches)
	i := 0
	for _, b := range branches {
		i = i + 1
		me.Logger.Info(fmt.Sprintf("Merging %d of %d (%s - %s)", i, cnt, b.Name, b.Label))
		err := me.MergeBranch(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (me *MergeExp) MergeBranch(b Branch) error {
	message := fmt.Sprintf("Experimental merge of %s", b.Label)
	err := me.Command("git", "merge", "--no-ff", "--log", "-m", message, b.Name).Run()
	if err != nil {
		return me.ResolveConflict(b, message)
	}

	return nil
}

func (me *MergeExp) ResolveConflict(b Branch, message string) error {
	var resolve func(int) error
	resolve = func(retry int) error {
		maxRetries := me.ConflictRetries
		if maxRetries == 0 {
			maxRetries = 4
		}
		if retry > maxRetries {
			return fmt.Errorf(fmt.Sprintf("Even after %d attempts the working dir is still not clean, aborting", retry))
		}

		hasunmerged := me.Command("git", "diff", "--exit-code", "--quiet", "--diff-filter=U").Run() != nil
		if hasunmerged {
			// are there any unmerged files (--diff-filter=U)
			output, _ := me.Command("git", "diff", "--name-only", "--diff-filter=U").Output()
			// lines := splitLines( output )

			me.Logger.Info(fmt.Sprintf(
				"Conflict in %s, you have unmerged files:\n%s\nResolve conflict, commit (or just add the files) and exit the shell",
				b.Name,
				string(output),
			))

			retryStr := ""
			if retry > 0 {
				retryStr = fmt.Sprintf(" (retry %d of %d)", retry, maxRetries)
			}

			prompt := fmt.Sprintf("Conflict resolution of '%s'%s $ ",
				b.Name,
				retryStr,
			)
			err := me.RunBashWithPrompt(prompt)
			if err != nil {
				return err
			}
			return resolve(retry + 1)
		} else {
			// no conflict - do we have some staged files
			hascached := me.Command("git", "diff", "--cached", "--exit-code", "--quiet").Run() != nil
			if hascached {
				newMessage := message
				if retry > 0 {
					newMessage = newMessage + " with resolved conflict(s) using rerere"
				}
				return me.Command("git", "commit", "-m", newMessage).Run()
			}
			return nil
		}
	}
	return resolve(0)
}

func (me *MergeExp) RunBashWithPrompt(prompt string) error {

	tmpFile, err := ioutil.TempFile(os.TempDir(), "mergeex*.sh")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}

	// Remember to clean up the file afterwards
	// defer os.Remove(tmpFile.Name())

	// Example writing to the file
	text := []byte(fmt.Sprintf("PS1='%s'\n", prompt))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	cmd := me.Command("bash", "--init-file", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (me *MergeExp) FinalCommit(remoteBranch *Branch) error {
	var err error
	var commitsNotIncluded string

	// test if remote branch exists
    remoteExists := me.Command("git", "show-branch", remoteBranch.Name).Run() == nil
    if ! remoteExists {
        return nil
    }
	if remoteExists {
		// remote branch exists
		commitsNotIncluded, err = OutputString(
			me.Command("git", "log", "--format=%h %ad %an%n     %s", "--no-merges", remoteBranch.Name+".."))
		if err != nil {
			return err
		}
	} else {
		commitsNotIncluded = fmt.Sprintf("Differential commits cannot be found, %s  does not exist so far", remoteBranch.Name)
	}

	message := fmt.Sprintf("Experimental merge")
	if true { 
        // OCP specific :-(
		message = message + " NOTESTS"
	}
	message = message + "\n"

	outstr, err := OutputString(me.Command("git", "log", "--oneline", "--first-parent", remoteBranch.Name+".."))
	if err != nil {
		return err
	}

	message = message + outstr + "\n\n" +
		fmt.Sprintf("Commit(s) included in this merge not present in last %s branch:",
			remoteBranch.Name,
		) +
		"\n\n" + commitsNotIncluded

	err = me.Command("git", "commit", "--allow-empty", "--message", message).Run()
	if err != nil {
		return err
	}
	// $this->call_system("git log -1");
	return nil
}
