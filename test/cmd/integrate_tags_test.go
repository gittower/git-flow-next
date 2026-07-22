package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestIntegrateWithTag verifies --tag creates an annotated tag on the parent tip.
//
// Steps:
//  1. init --defaults; add commit C to develop.
//  2. Run: git flow integrate develop --tag v2.0.0.
//  3. Assert main advanced to C and annotated tag v2.0.0 points at main's tip.
func TestIntegrateWithTag(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0")
	if err != nil {
		t.Fatalf("integrate develop --tag failed: %v\nOutput: %s", err, out)
	}

	if got := integRevParse(t, dir, "main"); got != cCommit {
		t.Errorf("Expected main to advance to C (%s), got %s", cCommit, got)
	}
	if !integTagExists(t, dir, "v2.0.0") {
		t.Fatal("Expected tag v2.0.0 to exist")
	}
	if !integTagIsAnnotated(t, dir, "v2.0.0") {
		t.Error("Expected v2.0.0 to be an annotated tag")
	}
	if integRevParse(t, dir, "v2.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v2.0.0 to point at main's tip")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateSignedTag verifies --sign produces a GPG-signed annotated tag.
// The test is skipped when GPG is unavailable or key generation fails.
//
// Steps:
//  1. init --defaults; add commit C to develop; configure an ephemeral GPG key.
//  2. Run: git flow integrate develop --tag v2.0.0 --sign.
//  3. Assert v2.0.0 is a signed annotated tag on main's tip.
func TestIntegrateSignedTag(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not available; skipping signed tag test")
	}

	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	// Ephemeral GPG home so we never touch the developer's keyring.
	gnupgHome := t.TempDir()
	if err := os.Chmod(gnupgHome, 0700); err != nil {
		t.Skipf("cannot secure GNUPGHOME: %v", err)
	}
	prev, hadPrev := os.LookupEnv("GNUPGHOME")
	os.Setenv("GNUPGHOME", gnupgHome)
	defer func() {
		if hadPrev {
			os.Setenv("GNUPGHOME", prev)
		} else {
			os.Unsetenv("GNUPGHOME")
		}
	}()

	genCmd := exec.Command("gpg", "--batch", "--pinentry-mode", "loopback", "--passphrase", "",
		"--quick-generate-key", "Test Signer <test@example.com>", "default", "default", "0")
	genCmd.Env = append(os.Environ(), "GNUPGHOME="+gnupgHome)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Skipf("gpg key generation failed, skipping: %v\nOutput: %s", err, out)
	}

	if _, err := testutil.RunGit(t, dir, "config", "user.signingkey", "test@example.com"); err != nil {
		t.Fatalf("Failed to configure signing key: %v", err)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0", "--sign")
	if err != nil {
		t.Fatalf("integrate develop --tag --sign failed: %v\nOutput: %s", err, out)
	}

	if !integTagExists(t, dir, "v2.0.0") {
		t.Fatal("Expected tag v2.0.0 to exist")
	}
	tagObj, err := testutil.RunGit(t, dir, "cat-file", "-p", "v2.0.0")
	if err != nil {
		t.Fatalf("Failed to read tag object: %v", err)
	}
	if !strings.Contains(tagObj, "-----BEGIN PGP SIGNATURE-----") {
		t.Errorf("Expected v2.0.0 to carry a PGP signature, got:\n%s", tagObj)
	}
	if integRevParse(t, dir, "v2.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v2.0.0 to point at main's tip")
	}
}

// TestIntegrateNotagOverridesConfig verifies --notag suppresses a configured tag.
//
// Steps:
//  1. init --defaults; add C to develop; set integrate.tag true and
//     integrate.tagname v9.
//  2. Run: git flow integrate develop --notag.
//  3. Assert main advanced to C but no tag was created.
func TestIntegrateNotagOverridesConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tag", "true")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tagname", "v9")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--notag")
	if err != nil {
		t.Fatalf("integrate develop --notag failed: %v\nOutput: %s", err, out)
	}

	if got := integRevParse(t, dir, "main"); got != cCommit {
		t.Errorf("Expected main to advance to C (%s), got %s", cCommit, got)
	}
	if tags, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "" {
		t.Errorf("Expected no tag with --notag, got: %s", tags)
	}
}

// TestIntegrateTagEnabledNoNameErrors verifies tagging enabled by config without
// a resolvable name errors before merging.
//
// Steps:
//  1. init --defaults; add C to develop; set integrate.tag true (no tagname, no --tag).
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit, main unchanged, no tag created.
func TestIntegrateTagEnabledNoNameErrors(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	preMain := integRevParse(t, dir, "main")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tag", "true")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate to fail when tagging is enabled without a name.\nOutput: %s", out)
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged before the error (%s), got %s", preMain, got)
	}
	if tags, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "" {
		t.Errorf("Expected no tag, got: %s", tags)
	}
}

// TestIntegrateMessagefileWinsOverMessage verifies --messagefile takes precedence
// over --message for the tag message.
//
// Steps:
//  1. init --defaults; add C to develop; write msg.txt with distinct content.
//  2. Run: git flow integrate develop --tag v2.0.0 --message "inline"
//     --messagefile msg.txt.
//  3. Assert the tag message comes from msg.txt, not "inline".
func TestIntegrateMessagefileWinsOverMessage(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	msgPath := filepath.Join(dir, "msg.txt")
	if err := os.WriteFile(msgPath, []byte("message from file"), 0644); err != nil {
		t.Fatalf("Failed to write msg.txt: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0",
		"--message", "inline", "--messagefile", msgPath)
	if err != nil {
		t.Fatalf("integrate with message + messagefile failed: %v\nOutput: %s", err, out)
	}

	msg := integTagMessage(t, dir, "v2.0.0")
	if !strings.Contains(msg, "message from file") {
		t.Errorf("Expected tag message from file, got: %s", msg)
	}
	if strings.Contains(msg, "inline") {
		t.Errorf("Expected --messagefile to win over --message, got: %s", msg)
	}
}

// TestIntegrateLayer2TagCreatesConfiguredTag verifies Layer-2 tag config alone
// creates the configured tag.
//
// Steps:
//  1. init --defaults; add C to develop; set integrate.tag true and
//     integrate.tagname v3.0.0 (no --tag/--notag).
//  2. Run: git flow integrate develop.
//  3. Assert annotated tag v3.0.0 on main's tip.
func TestIntegrateLayer2TagCreatesConfiguredTag(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tag", "true")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tagname", "v3.0.0")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	if !integTagExists(t, dir, "v3.0.0") {
		t.Fatal("Expected configured tag v3.0.0 to be created")
	}
	if !integTagIsAnnotated(t, dir, "v3.0.0") {
		t.Error("Expected v3.0.0 to be an annotated tag")
	}
	if integRevParse(t, dir, "v3.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v3.0.0 to point at main's tip")
	}
}

// TestIntegrateCliTagNameOverridesConfig verifies a CLI --tag name overrides a
// configured integrate.tagname.
//
// Steps:
//  1. init --defaults; add C to develop; set integrate.tag true and
//     integrate.tagname v3.0.0.
//  2. Run: git flow integrate develop --tag v4.0.0.
//  3. Assert v4.0.0 exists on main and configured v3.0.0 does not.
func TestIntegrateCliTagNameOverridesConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tag", "true")
	testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.tagname", "v3.0.0")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v4.0.0")
	if err != nil {
		t.Fatalf("integrate develop --tag v4.0.0 failed: %v\nOutput: %s", err, out)
	}

	if !integTagExists(t, dir, "v4.0.0") {
		t.Error("Expected CLI tag v4.0.0 to exist")
	}
	if integTagExists(t, dir, "v3.0.0") {
		t.Error("Expected configured tag v3.0.0 to be absent when CLI --tag wins")
	}
	if integRevParse(t, dir, "v4.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v4.0.0 to point at main's tip")
	}
}

// TestIntegrateMessageSetsTagMessage verifies a lone --message reaches the tag.
//
// Steps:
//  1. init --defaults; add C to develop.
//  2. Run: git flow integrate develop --tag v2.0.0 --message "integrate note".
//  3. Assert the tag message contains "integrate note".
func TestIntegrateMessageSetsTagMessage(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0", "--message", "integrate note")
	if err != nil {
		t.Fatalf("integrate develop --tag --message failed: %v\nOutput: %s", err, out)
	}

	msg := integTagMessage(t, dir, "v2.0.0")
	if !strings.Contains(msg, "integrate note") {
		t.Errorf("Expected tag message to contain 'integrate note', got: %s", msg)
	}
}
