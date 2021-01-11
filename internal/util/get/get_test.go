// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package get_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	err := Command{}.Run()
	assert.EqualError(t, err, "must specify repo")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	err := Command{Git: kptfile.Git{Repo: "foo"}}.Run()
	assert.EqualError(t, err, "must specify ref")
}

// TestCommand_Run verifies that Command will clone the HEAD of the master branch.
//
// - destination directory should match the base name of the repo
// - KptFile should be populated with values pointing to the origin
func TestCommand_Run(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{Git: kptfile.Git{
		Repo:      "file://" + g.RepoDirectory,
		Ref:       "master",
		Directory: "/",
	},
		Destination: absPath}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_subdir verifies that Command will clone a subdirectory of a repo.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
func TestCommand_Run_subdir(t *testing.T) {
	subdir := "java"
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, subdir)
	err := Command{Git: kptfile.Git{
		Repo: g.RepoDirectory, Ref: "refs/heads/master", Directory: subdir},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: subdir,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
		},
	})
}

// TestCommand_Run_destination verifies Command clones the repo to a destination with a specific name rather
// than using the name of the source repo.
func TestCommand_Run_destination(t *testing.T) {
	dest := "my-dataset"
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit,
			},
		},
	})
}

// TestCommand_Run_subdirAndDestination verifies that Command will copy a subdirectory of a repo to a
// specific destination.
//
// - name of the destination is used over the name of the subdir in the KptFile
func TestCommand_Run_subdirAndDestination(t *testing.T) {
	subdir := "java"
	dest := "new-java"
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: subdir,
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Commit:    commit,
				Directory: subdir,
				Ref:       "master",
				Repo:      g.RepoDirectory,
			},
		},
	})
}

// TestCommand_Run_branch verifies Command can clone a git branch
//
// 1. create a new branch
// 2. add data to the branch
// 3. checkout the master branch again
// 4. clone the new branch
// 5. verify contents match the new branch
func TestCommand_Run_branch(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// add commits to the exp branch
	err := g.CheckoutBranch("exp", true)
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new dataset")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.CheckoutBranch("master", false)
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit2)

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/heads/exp",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
				Commit:    commit,
			},
		},
	})
}

// TestCommand_Run_tag verifies Command can clone from a git tag
//
// 1. add data to the master branch
// 2. commit and tag the master branch
// 3. add more data to the master branch, commit it
// 4. clone at the tag
// 5. verify the clone has the data from the tagged version
func TestCommand_Run_tag(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset3)
	assert.NoError(t, err)
	err = g.Commit("new-data post-v2")
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)
	assert.NotEqual(t, commit, commit2)

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/tags/v2",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
	})
}

// TestCommand_Run_clean verifies that the Command delete the existing directory if Clean is set.
//
// 1. clone the master branch
// 2. add data to the master branch and commit it
// 3. clone the master branch again
// 4. verify the new master branch data is present
func TestCommand_Run_clean(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)

	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data")
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err = g.GetCommit()
	assert.NoError(t, err)

	// configure clone to clean the existing dir
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
		Clean:       true,
	}.Run()
	assert.NoError(t, err)

	// verify files are updated
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_failClean verifies that the Command will not clean the existing directory if it
// fails to clone.
//
// 1. clone the master branch
// 2. clone a non-existing branch
// 3. verify the master branch data is still present
func TestCommand_Run_failClean(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// configure clone to clean the existing dir, but fail
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/heads/not-real",
			Directory: "/",
		},
		Destination: absPath,
		Clean:       true,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/heads/not-real") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}

	// verify files weren't deleted
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_failExistingDir verifies that command will fail without changing anything if the
// directory already exists
func TestCommand_Run_failExistingDir(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data")
	assert.NoError(t, err)

	// try to clone and expect a failure
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.EqualError(t, err, fmt.Sprintf("destination directory %s already exists", absPath))

	// verify files are unchanged
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, "foo")
	err := Command{
		Git: kptfile.Git{
			Repo:      "foo",
			Directory: "/",
			Ref:       "refs/heads/master",
		},
		Destination: absPath,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "failed to lookup master(or main) branch") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidBranch(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoDirectory)
	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/heads/foo",
		},
		Destination: absPath,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/heads/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidTag(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	err := Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/tags/foo",
		},
		Destination: filepath.Join(w.WorkspaceDirectory, g.RepoDirectory),
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/tags/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}
}

func TestCommand_DefaultValues_AtVersion(t *testing.T) {
	c := Command{Git: kptfile.Git{Repo: "foo", Directory: "/", Ref: "r"}, Destination: "/"}
	assert.NoError(t, c.DefaultValues())

	c = Command{Git: kptfile.Git{Repo: "foo", Directory: "bar"}, Destination: "/"}
	assert.EqualError(t, c.DefaultValues(), "must specify ref")

	c = Command{Git: kptfile.Git{Ref: "foo", Repo: "bar"}, Destination: "/"}
	assert.EqualError(t, c.DefaultValues(), "must specify remote subdirectory")

	c = Command{Git: kptfile.Git{Ref: "foo", Directory: "bar"}, Destination: "/"}
	assert.EqualError(t, c.DefaultValues(), "must specify repo")

	c = Command{Git: kptfile.Git{Repo: "foo", Directory: "/", Ref: "r"}}
	assert.EqualError(t, c.DefaultValues(), "must specify destination")
}

func TestCommand_Run_subpackages(t *testing.T) {
	testCases := []struct {
		name            string
		directory       string
		ref             string
		upstream        testutil.Content
		refRepos        map[string][]testutil.Content
		expectedResults *pkgbuilder.RootPkg
	}{
		{
			name:      "basic package",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.DeploymentResource),
			},
			expectedResults: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
		},
		{
			name:      "package with subpackages",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expectedResults: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		{
			name:      "package with remote subpackages",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/", "master", "fast-forward"),
							),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithSubpackages(
												pkgbuilder.NewSubpackage("bar", "/foobar", "v1.2", "fast-forward"),
											),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResults: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "/", "master", "fast-forward"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			refRepos, err := testutil.SetupRepos(t, test.refRepos)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			repoPaths := make(map[string]string)
			for name, tgr := range refRepos {
				repoPaths[name] = tgr.RepoDirectory
			}

			dir := pkgbuilder.ExpandPkg(t, pkgbuilder.NewRootPkg(), repoPaths)
			g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
				Data:   dir,
				Branch: "master",
			}, repoPaths)
			defer clean()

			err = testutil.UpdateGitDir(t, g, []testutil.Content{test.upstream}, repoPaths)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var targetDir string
			if test.directory == "/" {
				targetDir = filepath.Base(g.RepoName)
			} else {
				targetDir = filepath.Base(test.directory)
			}
			w.PackageDir = targetDir
			destinationDir := filepath.Join(w.WorkspaceDirectory, targetDir)

			err = Command{
				Git: kptfile.Git{
					Repo:      g.RepoDirectory,
					Directory: test.directory,
					Ref:       test.ref,
				},
				Destination: destinationDir,
			}.Run()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := pkgbuilder.ExpandPkg(t, test.expectedResults, repoPaths)
			testutil.AssertPkgEqual(t, expectedPath, w.FullPackagePath())
		})
	}
}