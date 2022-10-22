package plan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmwave/helmwave/pkg/helper"
	"github.com/helmwave/helmwave/pkg/parallel"
	dir "github.com/otiai10/copy"
)

// Export allows save plan to file.
func (p *Plan) Export(ctx context.Context) error {
	if err := os.RemoveAll(p.dir); err != nil {
		return fmt.Errorf("failed to clean plan directory %s: %w", p.dir, err)
	}

	wg := parallel.NewWaitGroup()
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := p.exportManifest(); err != nil {
			wg.ErrChan() <- err
		}
	}()
	go func() {
		defer wg.Done()
		if err := p.exportValues(); err != nil {
			wg.ErrChan() <- err
		}

		// Save Planfile after values
		if err := helper.SaveInterface(ctx, p.fullPath, p.body); err != nil {
			wg.ErrChan() <- err
		}
	}()
	go func() {
		defer wg.Done()
		if err := p.exportGraphMD(); err != nil {
			wg.ErrChan() <- err
		}
	}()

	return wg.Wait()
}

func (p *Plan) exportManifest() error {
	if len(p.body.Releases) == 0 {
		return nil
	}

	for k, v := range p.manifests {
		m := filepath.Join(p.dir, Manifest, string(k)+".yml")

		f, err := helper.CreateFile(m)
		if err != nil {
			return err
		}

		_, err = f.WriteString(v)
		if err != nil {
			return fmt.Errorf("failed to write manifest %s: %w", f.Name(), err)
		}

		err = f.Close()
		if err != nil {
			return fmt.Errorf("failed to close manifest %s: %w", f.Name(), err)
		}
	}

	return nil
}

func (p *Plan) exportGraphMD() error {
	if len(p.body.Releases) == 0 {
		return nil
	}

	found := false
	for _, rel := range p.body.Releases {
		if len(rel.DependsOn()) > 0 {
			found = true

			break
		}
	}

	if !found {
		return nil
	}

	const filename = "graph.md"
	f, err := helper.CreateFile(filepath.Join(p.dir, filename))
	if err != nil {
		return err
	}

	_, err = f.WriteString(p.graphMD)
	if err != nil {
		return fmt.Errorf("failed to write graph file %s: %w", f.Name(), err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close graph file %s: %w", f.Name(), err)
	}

	return nil
}

func (p *Plan) exportValues() error {
	if len(p.body.Releases) == 0 {
		return nil
	}

	found := false

	for i, rel := range p.body.Releases {
		for j := range p.body.Releases[i].Values() {
			found = true
			p.body.Releases[i].Values()[j].SetUniq(p.dir, rel.Uniq())
		}
	}

	if !found {
		return nil
	}

	// It doesnt work if workdir has been mounted.
	err := os.Rename(
		filepath.Join(p.tmpDir, Values),
		filepath.Join(p.dir, Values),
	)
	if err != nil {
		err = dir.Copy(
			filepath.Join(p.tmpDir, Values),
			filepath.Join(p.dir, Values),
		)
		if err != nil {
			return fmt.Errorf("failed to copy values from %s to %s: %w", p.tmpDir, p.dir, err)
		}

		return nil
	}

	return nil
}

// IsExist returns true if planfile exists.
func (p *Plan) IsExist() bool {
	return helper.IsExists(p.fullPath)
}

// IsManifestExist returns true if planfile exists.
func (p *Plan) IsManifestExist() bool {
	return helper.IsExists(filepath.Join(p.dir, Manifest))
}
