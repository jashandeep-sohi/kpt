// Copyright 2020 Google LLC
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

package pkgbuilder

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	kptfileutil "github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	deploymentResourceManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: myspace
  name: mysql-deployment
spec:
  replicas: 3
  foo: bar
  template:
    spec:
      containers:
      - name: mysql
        image: mysql:1.7.9
`

	configMapResourceManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap
data:
  foo: bar
`
)

var (
	DeploymentResource = "deployment"
	ConfigMapResource  = "configmap"
	resources          = map[string]resourceInfo{
		DeploymentResource: {
			filename: "deployment.yaml",
			manifest: deploymentResourceManifest,
		},
		ConfigMapResource: {
			filename: "configmap.yaml",
			manifest: configMapResourceManifest,
		},
	}
)

// Pkg represents a package that can be created on the file system
// by using the Build function
type pkg struct {
	Kptfile *Kptfile

	resources []resourceInfoWithSetters

	files map[string]string

	subPkgs []*SubPkg
}

// withKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (p *pkg) withKptfile(kf ...*Kptfile) {
	if len(kf) > 1 {
		panic("only 0 or 1 Kptfiles are allowed")
	}
	if len(kf) == 0 {
		p.Kptfile = NewKptfile()
	} else {
		p.Kptfile = kf[0]
	}
}

// withResource configures the package to include the provided resource
func (p *pkg) withResource(resourceName string, mutators ...yaml.Filter) {
	resourceInfo, ok := resources[resourceName]
	if !ok {
		panic(fmt.Errorf("unknown resource %s", resourceName))
	}
	p.resources = append(p.resources, resourceInfoWithSetters{
		resourceInfo: resourceInfo,
		setterRefs:   []SetterRef{},
		mutators:     mutators,
	})
}

// withResourceAndSetters configures the package to have the provided resource.
// It also allows for specifying setterRefs for the resource and a set of
// mutators that will update the content of the resource.
func (p *pkg) withResourceAndSetters(resourceName string, setterRefs []SetterRef, mutators ...yaml.Filter) {
	resourceInfo, ok := resources[resourceName]
	if !ok {
		panic(fmt.Errorf("unknown resource %s", resourceName))
	}
	p.resources = append(p.resources, resourceInfoWithSetters{
		resourceInfo: resourceInfo,
		setterRefs:   setterRefs,
		mutators:     mutators,
	})
}

// withFile configures the package to contain a file with the provided name
// and the given content.
func (p *pkg) withFile(name, content string) {
	p.files[name] = content
}

// withSubPackages adds the provided packages as subpackages to the current
// package
func (p *pkg) withSubPackages(ps ...*SubPkg) {
	p.subPkgs = append(p.subPkgs, ps...)
}

// RootPkg is a package without any parent package.
type RootPkg struct {
	pkg *pkg
}

// NewRootPkg creates a new package for testing.
func NewRootPkg() *RootPkg {
	return &RootPkg{
		pkg: &pkg{
			files: make(map[string]string),
		},
	}
}

// WithKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (rp *RootPkg) WithKptfile(kf ...*Kptfile) *RootPkg {
	rp.pkg.withKptfile(kf...)
	return rp
}

// HasKptfile tells whether the package contains a Kptfile.
func (rp *RootPkg) HasKptfile() bool {
	return rp.pkg.Kptfile != nil
}

// WithResource configures the package to include the provided resource
func (rp *RootPkg) WithResource(resourceName string, mutators ...yaml.Filter) *RootPkg {
	rp.pkg.withResource(resourceName, mutators...)
	return rp
}

// WithResourceAndSetters configures the package to have the provided resource.
// It also allows for specifying setterRefs for the resource and a set of
// mutators that will update the content of the resource.
func (rp *RootPkg) WithResourceAndSetters(resourceName string, setterRefs []SetterRef, mutators ...yaml.Filter) *RootPkg {
	rp.pkg.withResourceAndSetters(resourceName, setterRefs, mutators...)
	return rp
}

// WithFile configures the package to contain a file with the provided name
// and the given content.
func (rp *RootPkg) WithFile(name, content string) *RootPkg {
	rp.pkg.withFile(name, content)
	return rp
}

// WithSubPackages adds the provided packages as subpackages to the current
// package
func (rp *RootPkg) WithSubPackages(ps ...*SubPkg) *RootPkg {
	rp.pkg.withSubPackages(ps...)
	return rp
}

// Build outputs the current data structure as a set of (nested) package
// in the provided path.
func (rp *RootPkg) Build(path string, pkgName string) error {
	pkgPath := filepath.Join(path, pkgName)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}
	err = buildPkg(pkgPath, rp.pkg, pkgName)
	if err != nil {
		return err
	}
	for i := range rp.pkg.subPkgs {
		subPkg := rp.pkg.subPkgs[i]
		err := buildSubPkg(pkgPath, subPkg)
		if err != nil {
			return err
		}
	}
	return nil
}

// SubPkg is a subpackage, so it is contained inside another package. The
// name sets both the name of the directory in which the package is stored
// and the metadata.name field in the Kptfile (if there is one).
type SubPkg struct {
	pkg *pkg

	Name string
}

// NewSubPkg returns a new subpackage for testing.
func NewSubPkg(name string) *SubPkg {
	return &SubPkg{
		pkg: &pkg{
			files: make(map[string]string),
		},
		Name: name,
	}
}

// WithKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (sp *SubPkg) WithKptfile(kf ...*Kptfile) *SubPkg {
	sp.pkg.withKptfile(kf...)
	return sp
}

// WithResource configures the package to include the provided resource
func (sp *SubPkg) WithResource(resourceName string, mutators ...yaml.Filter) *SubPkg {
	sp.pkg.withResource(resourceName, mutators...)
	return sp
}

// WithResourceAndSetters configures the package to have the provided resource.
// It also allows for specifying setterRefs for the resource and a set of
// mutators that will update the content of the resource.
func (sp *SubPkg) WithResourceAndSetters(resourceName string, setterRefs []SetterRef, mutators ...yaml.Filter) *SubPkg {
	sp.pkg.withResourceAndSetters(resourceName, setterRefs, mutators...)
	return sp
}

// WithFile configures the package to contain a file with the provided name
// and the given content.
func (sp *SubPkg) WithFile(name, content string) *SubPkg {
	sp.pkg.withFile(name, content)
	return sp
}

// WithSubPackages adds the provided packages as subpackages to the current
// package
func (sp *SubPkg) WithSubPackages(ps ...*SubPkg) *SubPkg {
	sp.pkg.withSubPackages(ps...)
	return sp
}

// Kptfile represents the Kptfile of a package.
type Kptfile struct {
	Setters []Setter
	Repo    string
	Ref     string
}

func NewKptfile() *Kptfile {
	return &Kptfile{}
}

// WithUpstream adds information about the upstream information to the Kptfile.
// The upstream section of the Kptfile is only added if this information is
// provided.
func (k *Kptfile) WithUpstream(repo, ref string) *Kptfile {
	k.Repo = repo
	k.Ref = ref
	return k
}

// WithSetters adds information about the setters for a Kptfile.
func (k *Kptfile) WithSetters(setters ...Setter) *Kptfile {
	k.Setters = setters
	return k
}

// Setter contains the properties required for adding a setter to the
// Kptfile.
type Setter struct {
	Name  string
	Value string
	IsSet bool
}

// NewSetter creates a new setter that is not marked as set
func NewSetter(name, value string) Setter {
	return Setter{
		Name:  name,
		Value: value,
	}
}

// NewSetSetter creates a new setter that is marked as set.
func NewSetSetter(name, value string) Setter {
	return Setter{
		Name:  name,
		Value: value,
		IsSet: true,
	}
}

// SetterRef specifies the information for creating a new reference to
// a setter in a resource.
type SetterRef struct {
	Path []string
	Name string
}

// NewSetterRef creates a new setterRef with the given name and path.
func NewSetterRef(name string, path ...string) SetterRef {
	return SetterRef{
		Path: path,
		Name: name,
	}
}

type resourceInfo struct {
	filename string
	manifest string
}

type resourceInfoWithSetters struct {
	resourceInfo resourceInfo
	setterRefs   []SetterRef
	mutators     []yaml.Filter
}

func buildSubPkg(path string, pkg *SubPkg) error {
	pkgPath := filepath.Join(path, pkg.Name)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}
	err = buildPkg(pkgPath, pkg.pkg, pkg.Name)
	if err != nil {
		return err
	}
	for i := range pkg.pkg.subPkgs {
		subPkg := pkg.pkg.subPkgs[i]
		err := buildSubPkg(pkgPath, subPkg)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildPkg(pkgPath string, pkg *pkg, pkgName string) error {
	if pkg.Kptfile != nil {
		content := buildKptfile(pkg, pkgName)

		err := ioutil.WriteFile(filepath.Join(pkgPath, kptfileutil.KptFileName),
			[]byte(content), 0600)
		if err != nil {
			return err
		}
	}

	for _, ri := range pkg.resources {
		m := ri.resourceInfo.manifest
		r := yaml.MustParse(m)
		for _, setterRef := range ri.setterRefs {
			n, err := r.Pipe(yaml.PathGetter{
				Path: setterRef.Path,
			})
			if err != nil {
				return err
			}
			n.YNode().LineComment = fmt.Sprintf(`{"$openapi":"%s"}`, setterRef.Name)
		}

		for _, m := range ri.mutators {
			if err := r.PipeE(m); err != nil {
				return err
			}
		}

		filePath := filepath.Join(pkgPath, ri.resourceInfo.filename)
		err := ioutil.WriteFile(filePath, []byte(r.MustString()), 0600)
		if err != nil {
			return err
		}
	}

	for name, content := range pkg.files {
		filePath := filepath.Join(pkgPath, name)
		_, err := os.Stat(filePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("file %s already exists", name)
		}
		err = ioutil.WriteFile(filePath, []byte(content), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

var kptfileTemplate = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: {{.PkgName}}
{{- if gt (len .Pkg.Kptfile.Setters) 0 }}
openAPI:
  definitions:
{{- range .Pkg.Kptfile.Setters }}
    io.k8s.cli.setters.{{.Name}}:
      x-k8s-cli:
        setter:
          name: {{.Name}}
          value: {{.Value}}
{{- if eq .IsSet true }}
          isSet: true
{{- end }}
{{- end }}
{{- end }}
{{- if gt (len .Pkg.Kptfile.Repo) 0 }}
upstream:
  type: git
  git:
    ref: {{.Pkg.Kptfile.Ref}}
    repo: {{.Pkg.Kptfile.Repo}}
{{- end }}
`

func buildKptfile(pkg *pkg, pkgName string) string {
	tmpl, err := template.New("test").Parse(kptfileTemplate)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Pkg":     pkg,
		"PkgName": pkgName,
	})
	if err != nil {
		panic(err)
	}
	result := buf.String()
	return result
}

// ExpandPkg writes the provided package to disk. The name of the root package
// will just be set to "base".
func ExpandPkg(t *testing.T, pkg *RootPkg) string {
	return ExpandPkgWithName(t, pkg, "base")
}

// ExpandPkgWithName writes the provided package to disk and uses the given
// rootName to set the value of the package directory and the metadata.name
// field of the root package.
func ExpandPkgWithName(t *testing.T, pkg *RootPkg, rootName string) string {
	dir, err := ioutil.TempDir("", "test-kpt-builder-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = pkg.Build(dir, rootName)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return filepath.Join(dir, rootName)
}