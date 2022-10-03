package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmwave/helmwave/pkg/registry"
	"github.com/helmwave/helmwave/pkg/release"
	"github.com/helmwave/helmwave/pkg/release/uniqname"
	"github.com/helmwave/helmwave/pkg/repo"
	"github.com/helmwave/helmwave/pkg/version"
	"github.com/invopop/jsonschema"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// Dir is default directory for generated files.
	Dir = ".helmwave/"

	// File is default file name for planfile.
	File = "planfile"

	// Body is default file name for main config.
	Body = "helmwave.yml"

	// Manifest is default directory under Dir for manifests.
	Manifest = "manifest/"

	// Values is default directory for values.
	Values = "values/"
)

var (
	// ErrManifestDirNotFound is an error for nonexistent manifest dir.
	ErrManifestDirNotFound = errors.New(Manifest + " dir not found")

	// ErrManifestDirEmpty is an error for empty manifest dir.
	ErrManifestDirEmpty = errors.New(Manifest + " is empty")
)

// Plan contains full helmwave state.
type Plan struct {
	body     *planBody
	dir      string
	fullPath string

	tmpDir string

	manifests map[uniqname.UniqName]string

	graphMD string

	templater string
}

// NewAndImport wrapper for New and Import in one.
func NewAndImport(src string) (p *Plan, err error) {
	p = New(src)

	err = p.Import()
	if err != nil {
		return p, err
	}

	return p, nil
}

// Logger will pretty build log.Entry.
func (p *Plan) Logger() *log.Entry {
	a := make([]string, 0, len(p.body.Releases))
	for _, r := range p.body.Releases {
		a = append(a, string(r.Uniq()))
	}

	b := make([]string, 0, len(p.body.Repositories))
	for _, r := range p.body.Repositories {
		b = append(b, r.Name())
	}

	c := make([]string, 0, len(p.body.Registries))
	for _, r := range p.body.Registries {
		c = append(c, r.Host())
	}

	return log.WithFields(log.Fields{
		"releases":     a,
		"repositories": b,
		"registries":   c,
	})
}

//nolint:lll
type planBody struct {
	Project      string           `json:"project" jsonschema:"title=the project name,description=reserved for future,example=my-awesome-project"`
	Version      string           `json:"version" jsonschema:"title=version of helmwave,description=will check current version and project version,example=0.23.0,example=0.22.1"`
	Repositories repo.Configs     `json:"repositories" jsonschema:"title=repositories list,description=helm repositories"`
	Registries   registry.Configs `json:"registries" jsonschema:"title=registries list,description=helm OCI registries"`
	Releases     release.Configs  `json:"releases" jsonschema:"title=helm releases,description=what you wanna deploy"`
}

func GenSchema() *jsonschema.Schema {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true

	return r.Reflect(&planBody{})
}

// NewBody parses plan from file.
func NewBody(file string) (*planBody, error) {
	b := &planBody{
		Version: version.Version,
	}

	src, err := os.ReadFile(file)
	if err != nil {
		return b, fmt.Errorf("failed to read plan file %s: %w", file, err)
	}

	err = yaml.Unmarshal(src, b)
	if err != nil {
		return b, fmt.Errorf("failed to unmarshal YAML plan %s: %w", file, err)
	}

	// Setup dev version
	// if b.Version == "" {
	// 	 b.Version = version.Version
	// }

	err = b.Validate()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// New returns empty *Plan for provided directory.
func New(dir string) *Plan {
	plan := &Plan{
		tmpDir:    os.TempDir(),
		dir:       dir,
		fullPath:  filepath.Join(dir, File),
		manifests: make(map[uniqname.UniqName]string),
	}

	return plan
}
