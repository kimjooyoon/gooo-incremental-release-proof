package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("deterministic JSON encoding failed: %v", err))
	}
	return Digest(encoded)
}

func MerkleRoot(records []ReleaseRecord) string {
	if len(records) == 0 {
		return Digest([]byte("empty-release-set"))
	}
	level := make([]string, 0, len(records))
	for _, record := range records {
		level = append(level, Digest([]byte("leaf\x00"+canonicalRelease(record))))
	}
	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)
		for index := 0; index < len(level); index += 2 {
			right := level[index]
			if index+1 < len(level) {
				right = level[index+1]
			}
			next = append(next, Digest([]byte("node\x00"+level[index]+"\x00"+right)))
		}
		level = next
	}
	return level[0]
}

type lockPreimage struct {
	Schema             string `json:"schema"`
	CheckpointID       string `json:"checkpoint_id"`
	PreviousRootDigest string `json:"previous_root_digest"`
	MerkleRootDigest   string `json:"merkle_root_digest"`
	ReleaseIDs         []int64 `json:"release_ids"`
}

func LockRootDigest(checkpoint Checkpoint) string {
	ids := make([]int64, 0, len(checkpoint.Releases))
	for _, record := range checkpoint.Releases {
		ids = append(ids, record.ImmutableReleaseID)
	}
	return DigestJSON(lockPreimage{
		Schema: checkpoint.Schema, CheckpointID: checkpoint.CheckpointID,
		PreviousRootDigest: checkpoint.PreviousRootDigest,
		MerkleRootDigest: checkpoint.MerkleRootDigest, ReleaseIDs: ids,
	})
}

func canonicalRelease(record ReleaseRecord) string {
	values := []string{
		strconv.FormatInt(record.ImmutableReleaseID, 10), record.Repository, record.Tag,
		record.AnnotatedTagObjectSHA, record.PeeledCommitSHA,
		strconv.FormatInt(record.CI.RunID, 10), record.CI.RunSHA,
		strconv.FormatInt(record.CI.JobID, 10), strconv.FormatInt(record.CI.ArtifactID, 10), record.CI.ArtifactName,
		strconv.FormatInt(record.SourceAsset.ID, 10), record.SourceAsset.Name, strconv.FormatInt(record.SourceAsset.Size, 10), record.SourceAsset.Digest,
		strconv.FormatInt(record.EvidenceAsset.ID, 10), record.EvidenceAsset.Name, strconv.FormatInt(record.EvidenceAsset.Size, 10), record.EvidenceAsset.Digest,
	}
	return strings.Join(values, "\x1f")
}

func CheckpointDigestMatches(checkpoint Checkpoint) bool {
	return checkpoint.MerkleRootDigest == MerkleRoot(checkpoint.Releases) && checkpoint.LockRootDigest == LockRootDigest(checkpoint)
}
