package proof

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	Root         string
	MetaPath     string
	CorpusPath   string
	OutputPath   string
	Toolchain    string
	RunnerDigest string
	ContractDigest string
}

func LoadCorpus(path string) (Corpus, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Corpus{}, nil, err
	}
	var corpus Corpus
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		return Corpus{}, nil, err
	}
	if corpus.Schema != "gooo/incremental-release-proof/corpus/v1" || corpus.CorpusID == "" {
		return Corpus{}, nil, errors.New("corpus schema or id is invalid")
	}
	return corpus, raw, nil
}

func WriteJSON(path, repositoryRoot string, value any) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absoluteRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("generated output must be outside the input repository")
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(absolutePath, append(encoded, '\n'), 0o644)
}

func ensureEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("caller-owned output directory must be empty: %s", path)
	}
	return nil
}

func ValidateCheckpoint(checkpoint Checkpoint) error {
	if checkpoint.Schema != "gooo/incremental-release-proof/checkpoint/v1" || checkpoint.CheckpointID == "" || checkpoint.PreviousRootDigest == "" || len(checkpoint.Releases) == 0 {
		return errors.New("checkpoint identity or release set is incomplete")
	}
	if checkpoint.MerkleRootDigest == "" || checkpoint.LockRootDigest == "" {
		return errors.New("checkpoint root digests are missing")
	}
	if !CheckpointDigestMatches(checkpoint) {
		return errors.New("checkpoint Merkle or lock root digest contradicts its contents")
	}
	for _, release := range checkpoint.Releases {
		if err := ValidateRelease(release); err != nil {
			return err
		}
	}
	return nil
}

func ValidateRelease(release ReleaseRecord) error {
	if release.ImmutableReleaseID <= 0 || release.Repository == "" || release.Tag == "" || release.AnnotatedTagObjectSHA == "" || release.PeeledCommitSHA == "" {
		return errors.New("release immutable identity is incomplete")
	}
	if release.CI.RunID <= 0 || release.CI.RunSHA == "" || release.CI.JobID <= 0 || release.CI.ArtifactID <= 0 || release.CI.ArtifactName == "" {
		return errors.New("CI run/job/artifact identity is incomplete")
	}
	if err := ValidateAsset(release.SourceAsset); err != nil {
		return fmt.Errorf("source asset: %w", err)
	}
	if err := ValidateAsset(release.EvidenceAsset); err != nil {
		return fmt.Errorf("evidence asset: %w", err)
	}
	return nil
}

func ValidateAsset(asset Asset) error {
	if asset.ID <= 0 || asset.Name == "" || asset.Size <= 0 || !strings.HasPrefix(asset.Digest, "sha256:") {
		return errors.New("asset id, name, positive size, and sha256 digest are required")
	}
	return nil
}
