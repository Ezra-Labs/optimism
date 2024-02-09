package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/mocks"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func makeTestBlobSidecar(index uint64) (eth.IndexedBlobHash, *eth.BlobSidecar) {
	blob := kzg4844.Blob{}
	// make first byte of test blob match its index so we can easily verify if is returned in the
	// expected order
	blob[0] = byte(index)
	commit, _ := kzg4844.BlobToCommitment(blob)
	proof, _ := kzg4844.ComputeBlobProof(blob, commit)
	hash := eth.KZGToVersionedHash(commit)

	idh := eth.IndexedBlobHash{
		Index: index,
		Hash:  hash,
	}
	sidecar := eth.BlobSidecar{
		Index:         eth.Uint64String(index),
		Blob:          eth.Blob(blob),
		KZGCommitment: eth.Bytes48(commit),
		KZGProof:      eth.Bytes48(proof),
	}
	return idh, &sidecar
}

func TestBlobsFromSidecars(t *testing.T) {
	indices := []uint64{5, 7, 2}

	// blobs should be returned in order of their indices in the hashes array regardless
	// of the sidecar ordering
	index0, sidecar0 := makeTestBlobSidecar(indices[0])
	index1, sidecar1 := makeTestBlobSidecar(indices[1])
	index2, sidecar2 := makeTestBlobSidecar(indices[2])

	hashes := []eth.IndexedBlobHash{index0, index1, index2}

	// put the sidecars in scrambled order to confirm error
	sidecars := []*eth.BlobSidecar{sidecar2, sidecar0, sidecar1}
	_, err := blobsFromSidecars(sidecars, hashes)
	require.Error(t, err)

	// too few sidecars should error
	sidecars = []*eth.BlobSidecar{sidecar0, sidecar1}
	_, err = blobsFromSidecars(sidecars, hashes)
	require.Error(t, err)

	// correct order should work
	sidecars = []*eth.BlobSidecar{sidecar0, sidecar1, sidecar2}
	blobs, err := blobsFromSidecars(sidecars, hashes)
	require.NoError(t, err)
	// confirm order by checking first blob byte against expected index
	for i := range blobs {
		require.Equal(t, byte(indices[i]), blobs[i][0])
	}

	// mangle a proof to make sure it's detected
	badProof := *sidecar0
	badProof.KZGProof[11]++
	sidecars[1] = &badProof
	_, err = blobsFromSidecars(sidecars, hashes)
	require.Error(t, err)

	// mangle a commitment to make sure it's detected
	badCommitment := *sidecar0
	badCommitment.KZGCommitment[13]++
	sidecars[1] = &badCommitment
	_, err = blobsFromSidecars(sidecars, hashes)
	require.Error(t, err)

	// mangle a hash to make sure it's detected
	sidecars[1] = sidecar0
	hashes[2].Hash[17]++
	_, err = blobsFromSidecars(sidecars, hashes)
	require.Error(t, err)
}

func TestBlobsFromSidecars_EmptySidecarList(t *testing.T) {
	hashes := []eth.IndexedBlobHash{}
	sidecars := []*eth.BlobSidecar{}
	blobs, err := blobsFromSidecars(sidecars, hashes)
	require.NoError(t, err)
	require.Empty(t, blobs, "blobs should be empty when no sidecars are provided")
}

func TestFallbackClient(t *testing.T) {
	ctx := context.Background()
	a := mocks.NewBeaconClient(t)
	b := mocks.NewBeaconClient(t)
	cl := FallbackBeaconClient{primary: a, secondary: b, l: testlog.Logger(t, slog.LevelError)}

	// Test that it goes to the primary
	version := "mock 1.2.3"
	a.EXPECT().NodeVersion(ctx).Once().Return(version, nil)
	out, err := cl.NodeVersion(ctx)
	require.Equal(t, version, out)
	require.NoError(t, err)

	// Test that if falls back on error
	versionb := "mock archiver 2.3.4"
	a.EXPECT().NodeVersion(ctx).Once().Return("", errors.New("timeout"))
	b.EXPECT().NodeVersion(ctx).Once().Return(versionb, nil)
	out, err = cl.NodeVersion(ctx)
	require.Equal(t, versionb, out)
	require.NoError(t, err)

}
